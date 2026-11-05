// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbmodel "github.com/thunder-id/thunderid/internal/system/database/model"
	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

func columnRowsOf(columns ...columnSpec) []engine.QueryRow {
	rows := make([]engine.QueryRow, 0, len(columns))
	for _, column := range columns {
		rows = append(rows, engine.QueryRow{
			"column_name": column.name, "is_nullable": column.nullable, "data_type": column.dataType,
			"maximum_length": column.maximumLength, "column_default": column.defaultValue,
		})
	}
	return rows
}

func indexRowOf(index indexSpec) engine.QueryRow {
	return engine.QueryRow{"index_name": index.name, "is_unique": index.unique, "column_names": index.columns}
}

// existingTable answers the reads of an add-column operation for a table with some columns and indexes. A
// relation named in taken exists and is not the table.
func existingTable(table, kind string, columns []engine.QueryRow, indexes []engine.QueryRow, taken string,
) *fakeAccess {
	return newFakeAccess().
		onArgs(queryTableRelation, func(args []interface{}) []engine.QueryRow {
			name, _ := args[0].(string)
			switch {
			case name == table && kind != "":
				return []engine.QueryRow{{"relation_kind": kind}}
			case name == taken && taken != "":
				return []engine.QueryRow{{"relation_kind": "i"}}
			}
			return nil
		}).
		on(queryTableColumns, columns...).
		on(queryTableIndexes, indexes...)
}

func TestCreateGatewayMasterOperation(t *testing.T) {
	assertCreateTableOperation(t, newCreateGatewayMasterOperation, createGatewayMasterOperationID,
		gatewayMasterTable, 1)

	assert.Contains(t, postgresCreateGatewayMasterTableSQL, "ID VARCHAR(36) PRIMARY KEY")
	assert.Contains(t, postgresCreateGatewayMasterTableSQL, "UNIQUE (DEPLOYMENT_ID, GATEWAY_ID, SEQ)")

	// A table without its unique constraint is not the target table.
	operation, err := newCreateGatewayMasterOperation()
	require.NoError(t, err)
	changed := gatewayMasterTable
	changed.keys = []keySpec{{typeCode: "p", columns: "id"}}
	verification, err := operation.PostVerify(context.Background(), tableReadAccess{table: changed},
		engine.ExecutionTarget{})
	require.NoError(t, err)
	assert.Equal(t, engine.VerificationInvalid, verification.State)
}

func TestAddColumnOperationsShape(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() (engine.Operation, error)
		id          engine.OperationID
		store       engine.StoreID
		statements  []string
	}{
		{name: "gateway applied version", constructor: newAddGatewayAppliedVersionMasterColumnsOperation,
			id: addGatewayAppliedVersionMasterColumnsOperationID, store: engine.StoreIDConfigDB,
			statements: []string{"set-lock-timeout", "add-master-columns"}},
		{name: "entity project", constructor: newAddEntityProjectIDOperation,
			id: addEntityProjectIDOperationID, store: engine.StoreIDEntityDB,
			statements: []string{"set-lock-timeout", "add-project-id", "create-entity-project-index"}},
		{name: "organization unit project", constructor: newAddOrganizationUnitProjectIDOperation,
			id: addOrganizationUnitProjectIDOperationID, store: engine.StoreIDEntityDB,
			statements: []string{"set-lock-timeout", "add-project-id", "create-organization-unit-project-index"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation, err := test.constructor()
			require.NoError(t, err)
			definition := operation.Definition()
			assert.Equal(t, test.id, definition.ID)
			assert.Equal(t, test.store, definition.StoreID)
			assert.Equal(t, engine.OperationKindSchemaChange, definition.Kind)
			assert.Equal(t, engine.InterruptionModeAtomic, definition.InterruptionMode)

			implementation, err := operation.(*engine.SQLOperation).Implementation(engine.DatabaseTypePostgres)
			require.NoError(t, err)
			ids := make([]string, 0, len(implementation.Statements))
			for _, statement := range implementation.Statements {
				ids = append(ids, statement.ID)
			}
			assert.Equal(t, test.statements, ids)
			assert.Equal(t, "SET LOCAL lock_timeout = '2s'", implementation.Statements[0].Query)
		})
	}
}

func TestAddGatewayAppliedVersionMasterColumnsStatement(t *testing.T) {
	operation, err := newAddGatewayAppliedVersionMasterColumnsOperation()
	require.NoError(t, err)
	implementation, err := operation.(*engine.SQLOperation).Implementation(engine.DatabaseTypePostgres)
	require.NoError(t, err)
	query := implementation.Statements[1].Query
	for _, fragment := range []string{
		`ALTER TABLE "GATEWAY_APPLIED_VERSION"`, "ADD COLUMN COMPLETE BOOLEAN NOT NULL DEFAULT TRUE",
		"ADD COLUMN MASTER_SEQ INTEGER", "ADD COLUMN MASTER_HASH VARCHAR(64)",
		"ADD COLUMN PREVIOUS_MASTER_SEQ INTEGER", "ADD COLUMN RESOURCES_HELD INTEGER",
		"ADD COLUMN RESOURCES_TOTAL INTEGER",
	} {
		assert.Contains(t, query, fragment)
	}
}

func TestAddColumnsWithoutIndexesVerification(t *testing.T) {
	operation, err := newAddGatewayAppliedVersionMasterColumnsOperation()
	require.NoError(t, err)
	spec := []columnSpec{
		{"complete", "NO", "boolean", "none", "true"},
		{"master_seq", "YES", "integer", "none", "none"},
		{"master_hash", "YES", "character varying", "64", "none"},
		{"previous_master_seq", "YES", "integer", "none", "none"},
		{"resources_held", "YES", "integer", "none", "none"},
		{"resources_total", "YES", "integer", "none", "none"},
	}
	existing := columnSpec{"applied_seq", "NO", "integer", "none", "none"}
	wrongDefault := append([]columnSpec(nil), spec...)
	wrongDefault[0].defaultValue = "false"

	tests := []struct {
		name    string
		access  *fakeAccess
		state   engine.VerificationState
		summary string
	}{
		{name: "all absent", access: existingTable("GATEWAY_APPLIED_VERSION", "r", columnRowsOf(existing), nil, ""),
			state: engine.VerificationSource, summary: "GATEWAY_APPLIED_VERSION.master_seq, "},
		{name: "all present", access: existingTable("GATEWAY_APPLIED_VERSION", "r",
			columnRowsOf(append([]columnSpec{existing}, spec...)...), nil, ""),
			state: engine.VerificationTarget, summary: " are present"},
		{name: "some present", access: existingTable("GATEWAY_APPLIED_VERSION", "r",
			columnRowsOf(append([]columnSpec{existing}, spec[:3]...)...), nil, ""),
			state: engine.VerificationInvalid, summary: "partly present: 3 of 6"},
		{name: "a column with another default", access: existingTable("GATEWAY_APPLIED_VERSION", "r",
			columnRowsOf(append([]columnSpec{existing}, wrongDefault...)...), nil, ""),
			state: engine.VerificationInvalid, summary: "Unexpected GATEWAY_APPLIED_VERSION.complete definition"},
		{name: "table missing", access: existingTable("GATEWAY_APPLIED_VERSION", "", nil, nil, ""),
			state: engine.VerificationInvalid, summary: "Expected one GATEWAY_APPLIED_VERSION table"},
		{name: "table is a view", access: existingTable("GATEWAY_APPLIED_VERSION", "v", nil, nil, ""),
			state: engine.VerificationInvalid, summary: "not an ordinary table"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := operation.PreVerify(context.Background(), test.access, engine.ExecutionTarget{})
			require.NoError(t, err)
			assert.Equal(t, test.state, result.State, result.Summary)
			assert.Contains(t, result.Summary, test.summary)

			readiness, err := operation.(*engine.SQLOperation).CheckReadiness(
				context.Background(), test.access, engine.PlanContext{})
			require.NoError(t, err)
			if test.state == engine.VerificationInvalid {
				assert.Equal(t, engine.ReadinessBlocked, readiness.State)
			} else {
				assert.Equal(t, engine.ReadinessReady, readiness.State)
			}
		})
	}
}

func TestAddColumnWithIndexVerification(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() (engine.Operation, error)
		table       string
		column      columnSpec
		index       indexSpec
	}{
		{name: "entity", constructor: newAddEntityProjectIDOperation, table: "ENTITY",
			column: columnSpec{"project_id", "YES", "character varying", "36", "none"},
			index: indexSpec{name: "idx_entity_project", unique: "false",
				columns: "deployment_id,category,project_id"}},
		{name: "organization unit", constructor: newAddOrganizationUnitProjectIDOperation,
			table:  "ORGANIZATION_UNIT",
			column: columnSpec{"project_id", "YES", "character varying", "36", "none"},
			index:  indexSpec{name: "idx_ou_project", unique: "false", columns: "deployment_id,project_id"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation, err := test.constructor()
			require.NoError(t, err)
			base := columnSpec{"deployment_id", "NO", "character varying", "255", "none"}
			other := indexSpec{name: "idx_other", unique: "false", columns: "deployment_id"}
			longer := test.column
			longer.maximumLength = "100"
			changedIndex := test.index
			changedIndex.unique = "true"

			cases := []struct {
				name    string
				access  *fakeAccess
				state   engine.VerificationState
				summary string
			}{
				{name: "column and index absent", access: existingTable(test.table, "r",
					columnRowsOf(base), []engine.QueryRow{indexRowOf(other)}, ""),
					state: engine.VerificationSource},
				{name: "column and index present", access: existingTable(test.table, "r",
					columnRowsOf(base, test.column), []engine.QueryRow{indexRowOf(other), indexRowOf(test.index)}, ""),
					state: engine.VerificationTarget},
				{name: "column without its index", access: existingTable(test.table, "r",
					columnRowsOf(base, test.column), []engine.QueryRow{indexRowOf(other)}, ""),
					state: engine.VerificationInvalid, summary: "partly present: 1 of 2"},
				{name: "index without its column", access: existingTable(test.table, "r",
					columnRowsOf(base), []engine.QueryRow{indexRowOf(test.index)}, ""),
					state: engine.VerificationInvalid, summary: "partly present: 1 of 2"},
				{name: "column of another length", access: existingTable(test.table, "r",
					columnRowsOf(base, longer), []engine.QueryRow{indexRowOf(test.index)}, ""),
					state: engine.VerificationInvalid, summary: "Unexpected " + test.table + ".project_id definition"},
				{name: "index with another definition", access: existingTable(test.table, "r",
					columnRowsOf(base, test.column), []engine.QueryRow{indexRowOf(changedIndex)}, ""),
					state: engine.VerificationInvalid, summary: "unexpected uniqueness or columns"},
				{name: "index name used by another relation", access: existingTable(test.table, "r",
					columnRowsOf(base), []engine.QueryRow{indexRowOf(other)}, test.index.name),
					state: engine.VerificationInvalid, summary: "already in use"},
			}
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					result, err := operation.PreVerify(context.Background(), c.access, engine.ExecutionTarget{})
					require.NoError(t, err)
					assert.Equal(t, c.state, result.State, result.Summary)
					assert.Contains(t, result.Summary, c.summary)
				})
			}
		})
	}
}

func TestAddColumnPropagatesInspectionErrors(t *testing.T) {
	operation, err := newAddEntityProjectIDOperation()
	require.NoError(t, err)
	access := existingTable("ENTITY", "r", columnRowsOf(), nil, "")
	access.err = assert.AnError
	_, err = operation.PreVerify(context.Background(), access, engine.ExecutionTarget{})
	require.ErrorIs(t, err, assert.AnError)
}

func statementIDs(t *testing.T, operation engine.Operation) []string {
	t.Helper()
	implementation, err := operation.(*engine.SQLOperation).Implementation(engine.DatabaseTypePostgres)
	require.NoError(t, err)
	ids := make([]string, 0, len(implementation.Statements))
	for _, statement := range implementation.Statements {
		ids = append(ids, statement.ID)
	}
	return ids
}

func TestReplaceProjectTableOperation(t *testing.T) {
	operation, err := newReplaceProjectTableOperation()
	require.NoError(t, err)
	definition := operation.Definition()
	assert.Equal(t, replaceProjectTableOperationID, definition.ID)
	assert.Equal(t, engine.StoreIDEntityDB, definition.StoreID)
	assert.Equal(t, engine.OperationKindSchemaChange, definition.Kind)
	assert.Equal(t, engine.InterruptionModeAtomic, definition.InterruptionMode)

	assert.Equal(t, []string{"set-lock-timeout", "rename-project", "rename-project-legacy-primary-key",
		"create-project", "create-project-handle-index"}, statementIDs(t, operation))
	implementation, err := operation.(*engine.SQLOperation).Implementation(engine.DatabaseTypePostgres)
	require.NoError(t, err)
	assert.Contains(t, implementation.Statements[1].Query, `RENAME TO "PROJECT_LEGACY"`)
	assert.Contains(t, implementation.Statements[2].Query, "PROJECT_LEGACY_pkey")
	assert.Contains(t, implementation.Statements[3].Query, "HANDLE          VARCHAR(100) NOT NULL")
	assert.Contains(t, implementation.Statements[4].Query, "CREATE UNIQUE INDEX idx_project_handle")
	for _, statement := range implementation.Statements {
		assert.NotContains(t, statement.Query, "DROP TABLE", "nothing is dropped while the table is replaced")
	}
}

func TestReplaceProjectTableVerification(t *testing.T) {
	legacy := func(kind string, taken ...string) *fakeAccess {
		takenNames := map[string]bool{}
		for _, name := range taken {
			takenNames[name] = true
		}
		return newFakeAccess().
			onArgs(queryTableRelation, func(args []interface{}) []engine.QueryRow {
				name, _ := args[0].(string)
				switch {
				case name == "PROJECT" && kind != "":
					return []engine.QueryRow{{"relation_kind": kind}}
				case takenNames[name]:
					return []engine.QueryRow{{"relation_kind": "r"}}
				}
				return nil
			}).
			on(queryTableColumns, columnRowsOf(legacyProjectColumns...)...)
	}

	tests := []struct {
		name    string
		access  engine.ReadAccess
		state   engine.VerificationState
		summary string
	}{
		{name: "legacy table", access: legacy("r"), state: engine.VerificationSource,
			summary: "ready to be kept as PROJECT_LEGACY"},
		{name: "target table", access: tableReadAccess{table: projectTable}, state: engine.VerificationTarget,
			summary: "already matches"},
		{name: "PROJECT_LEGACY already exists", access: legacy("r", "PROJECT_LEGACY"),
			state: engine.VerificationInvalid, summary: "Relation name PROJECT_LEGACY is already in use"},
		{name: "the handle index name is taken", access: legacy("r", "idx_project_handle"),
			state: engine.VerificationInvalid, summary: "Relation name idx_project_handle is already in use"},
		{name: "another shape", access: newFakeAccess().relations("PROJECT").
			on(queryTableColumns, columnRowsOf(legacyProjectColumns[:3]...)...), state: engine.VerificationInvalid,
			summary: "neither the previous release's schema nor the target schema"},
		{name: "table missing", access: legacy(""), state: engine.VerificationInvalid,
			summary: "Expected one PROJECT table"},
		{name: "table is a view", access: legacy("v"), state: engine.VerificationInvalid,
			summary: "not an ordinary table"},
	}
	operation, err := newReplaceProjectTableOperation()
	require.NoError(t, err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := operation.PreVerify(context.Background(), test.access, engine.ExecutionTarget{})
			require.NoError(t, err)
			assert.Equal(t, test.state, result.State, result.Summary)
			assert.Contains(t, result.Summary, test.summary)

			readiness, err := operation.(*engine.SQLOperation).CheckReadiness(
				context.Background(), test.access, engine.PlanContext{})
			require.NoError(t, err)
			if test.state == engine.VerificationInvalid {
				assert.Equal(t, engine.ReadinessBlocked, readiness.State)
				assert.Len(t, readiness.Findings, 1)
			} else {
				assert.Equal(t, engine.ReadinessReady, readiness.State)
			}
		})
	}
}

func TestDropProjectLegacyTableOperation(t *testing.T) {
	operation, err := newDropProjectLegacyTableOperation()
	require.NoError(t, err)
	definition := operation.Definition()
	assert.Equal(t, dropProjectLegacyTableOperationID, definition.ID)
	assert.Equal(t, engine.StoreIDEntityDB, definition.StoreID)
	assert.Equal(t, engine.OperationKindSchemaChange, definition.Kind)
	assert.Equal(t, engine.InterruptionModeAtomic, definition.InterruptionMode)

	assert.Equal(t, []string{"set-lock-timeout", "require-empty-project-legacy", "drop-project-legacy"},
		statementIDs(t, operation))
	implementation, err := operation.(*engine.SQLOperation).Implementation(engine.DatabaseTypePostgres)
	require.NoError(t, err)
	assert.Contains(t, implementation.Statements[1].Query, "to_regclass")
	assert.Contains(t, implementation.Statements[1].Query, "RAISE EXCEPTION")
	assert.Equal(t, `DROP TABLE IF EXISTS "PROJECT_LEGACY"`, implementation.Statements[2].Query)
}

func TestDropProjectLegacyTableVerification(t *testing.T) {
	access := func(legacyKind string, projectIsTarget bool, projects int64) engine.ReadAccess {
		return projectLegacyAccess{
			legacyKind: legacyKind, projectIsTarget: projectIsTarget, projects: projects,
		}
	}
	tests := []struct {
		name    string
		access  engine.ReadAccess
		state   engine.VerificationState
		summary string
	}{
		{name: "already dropped", access: access("", true, 0), state: engine.VerificationTarget,
			summary: "PROJECT_LEGACY is absent"},
		{name: "empty table", access: access("r", true, 0), state: engine.VerificationSource,
			summary: "empty and ready to be dropped"},
		{name: "table with projects", access: access("r", true, 3), state: engine.VerificationInvalid,
			summary: "holds 3 projects"},
		{name: "replacement not in place", access: access("r", false, 0), state: engine.VerificationInvalid,
			summary: "cannot be dropped before PROJECT has the target schema"},
		{name: "not an ordinary table", access: access("v", true, 0), state: engine.VerificationInvalid,
			summary: "not an ordinary table"},
	}
	operation, err := newDropProjectLegacyTableOperation()
	require.NoError(t, err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := operation.PreVerify(context.Background(), test.access, engine.ExecutionTarget{})
			require.NoError(t, err)
			assert.Equal(t, test.state, result.State, result.Summary)
			assert.Contains(t, result.Summary, test.summary)

			readiness, err := operation.(*engine.SQLOperation).CheckReadiness(
				context.Background(), test.access, engine.PlanContext{})
			require.NoError(t, err)
			if test.state == engine.VerificationInvalid {
				assert.Equal(t, engine.ReadinessBlocked, readiness.State)
			} else {
				assert.Equal(t, engine.ReadinessReady, readiness.State)
			}
		})
	}
}

func TestDropProjectLegacyTableRejectsAnUnreadableRowCount(t *testing.T) {
	operation, err := newDropProjectLegacyTableOperation()
	require.NoError(t, err)
	_, err = operation.PreVerify(context.Background(),
		projectLegacyAccess{legacyKind: "r", projectIsTarget: true, noCount: true}, engine.ExecutionTarget{})
	require.ErrorContains(t, err, "expected one row")
}

// projectLegacyAccess reports PROJECT_LEGACY as absent or as a table holding some projects, next to a PROJECT
// that either has the target shape or still has the legacy one.
type projectLegacyAccess struct {
	legacyKind      string
	projectIsTarget bool
	projects        int64
	noCount         bool
}

func (a projectLegacyAccess) Query(
	ctx context.Context,
	storeID engine.StoreID,
	query dbmodel.DBQuery,
	args ...interface{},
) ([]engine.QueryRow, error) {
	if query.GetID() == queryProjectLegacyRowCount.GetID() {
		if a.noCount {
			return nil, nil
		}
		return []engine.QueryRow{{"row_count": a.projects}}, nil
	}
	if query.GetID() == queryTableRelation.GetID() && args[0] == projectLegacyTable {
		if a.legacyKind == "" {
			return nil, nil
		}
		return []engine.QueryRow{{"relation_kind": a.legacyKind}}, nil
	}
	if a.projectIsTarget {
		return tableReadAccess{table: projectTable}.Query(ctx, storeID, query, args...)
	}
	switch query.GetID() {
	case queryTableRelation.GetID():
		return []engine.QueryRow{{"relation_kind": "r"}}, nil
	case queryTableColumns.GetID():
		return columnRowsOf(legacyProjectColumns...), nil
	}
	return nil, nil
}

func (projectLegacyAccess) DatabaseType(engine.StoreID) (engine.DatabaseType, error) {
	return engine.DatabaseTypePostgres, nil
}

// Every operation that creates a table looks for it first. It is skipped when the table is already there with the
// target shape, and blocked, with a finding, when something else already has the name or the table is partly
// there, so a create statement never runs against a relation it did not expect.
func TestEveryCreateTableOperationChecksForAnExistingTable(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() (engine.Operation, error)
		table       tableSpec
	}{
		{name: "SESSION", constructor: newCreateSessionOperation, table: sessionTable},
		{name: "SESSION_PARTICIPANT", constructor: newCreateSessionParticipantOperation,
			table: sessionParticipantTable},
		{name: "USER_SHARE_MIGRATION", constructor: newCreateUserShareMigrationTableOperation,
			table: userShareMigrationTable},
		{name: "GATEWAY_MASTER", constructor: newCreateGatewayMasterOperation, table: gatewayMasterTable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation, err := test.constructor()
			require.NoError(t, err)
			readiness := func(access tableReadAccess) engine.ReadinessResult {
				result, err := operation.(*engine.SQLOperation).CheckReadiness(
					context.Background(), access, engine.PlanContext{})
				require.NoError(t, err)
				return result
			}
			verify := func(access tableReadAccess) engine.VerificationResult {
				result, err := operation.PreVerify(context.Background(), access, engine.ExecutionTarget{})
				require.NoError(t, err)
				return result
			}

			// Absent: it will be created.
			absent := tableReadAccess{table: test.table, absent: true}
			assert.Equal(t, engine.VerificationSource, verify(absent).State)
			assert.Equal(t, engine.ReadinessReady, readiness(absent).State)

			// Already there as the target: nothing to create.
			present := tableReadAccess{table: test.table}
			assert.Equal(t, engine.VerificationTarget, verify(present).State)

			// The name belongs to something that is not a table: blocked.
			notATable := tableReadAccess{table: test.table, invalidType: true}
			assert.Equal(t, engine.VerificationInvalid, verify(notATable).State)
			blocked := readiness(notATable)
			assert.Equal(t, engine.ReadinessBlocked, blocked.State)
			assert.Len(t, blocked.Findings, 1)

			// A table of that name with another shape: blocked.
			changed := test.table.columns[0]
			changed.dataType = "text"
			other := tableReadAccess{table: test.table, columnOverride: map[int]columnSpec{0: changed}}
			assert.Equal(t, engine.VerificationInvalid, verify(other).State)
			assert.Equal(t, engine.ReadinessBlocked, readiness(other).State)
		})
	}
}

func TestAddColumnSummariesAgreeWithWhatTheyName(t *testing.T) {
	single, err := newAddInboundClientDeviceGrantFlowIDOperation()
	require.NoError(t, err)
	result, err := single.PreVerify(context.Background(), existingTable("INBOUND_CLIENT", "r",
		columnRowsOf(columnSpec{"auth_flow_id", "NO", "character varying", "100", "none"}), nil, ""),
		engine.ExecutionTarget{})
	require.NoError(t, err)
	assert.Equal(t, "INBOUND_CLIENT.device_grant_flow_id is absent", result.Summary)

	several, err := newAddEntityProjectIDOperation()
	require.NoError(t, err)
	result, err = several.PreVerify(context.Background(), existingTable("ENTITY", "r",
		columnRowsOf(columnSpec{"deployment_id", "NO", "character varying", "255", "none"}), nil, ""),
		engine.ExecutionTarget{})
	require.NoError(t, err)
	assert.Equal(t, "ENTITY.project_id, index idx_entity_project are absent", result.Summary)
}
