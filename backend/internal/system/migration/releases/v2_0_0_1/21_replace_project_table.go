// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"fmt"

	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

const (
	replaceProjectTableOperationID engine.OperationID = "2.0.0-m1-pidp/entitydb/21_replace_project_table"

	projectLegacyTable = "PROJECT_LEGACY"
)

// legacyProjectColumns is the PROJECT table of the previous release, where a project was only the root
// organization unit that backs it.
var legacyProjectColumns = []columnSpec{
	{"deployment_id", "NO", "character varying", "255", "none"},
	{"id", "NO", "character varying", "36", "none"},
	{"root_ou_id", "NO", "character varying", "36", "none"},
	{"created_at", "YES", "timestamp with time zone", "none", "now()"},
}

// projectTable is the PROJECT table of the target release: a project has its own handle, display name and
// description instead of borrowing them from a root organization unit.
var projectTable = tableSpec{
	storeID: engine.StoreIDEntityDB,
	name:    "PROJECT",
	columns: []columnSpec{
		{"deployment_id", "NO", "character varying", "255", "none"},
		{"id", "NO", "character varying", "36", "none"},
		{"handle", "NO", "character varying", "100", "none"},
		{"display_name", "NO", "character varying", "100", "none"},
		{"description", "YES", "character varying", "255", "none"},
		{"created_at", "YES", "timestamp with time zone", "none", "now()"},
	},
	keys: []keySpec{{typeCode: "p", columns: "deployment_id,id"}},
	indexes: []indexSpec{
		{name: "idx_project_handle", unique: "true", columns: "deployment_id,handle"},
	},
}

// newReplaceProjectTableOperation gives the project table its target shape without deleting anything.
//
// A legacy project is only a root organization unit, and the target needs a handle and a display name that
// nothing in a legacy row supplies, so the rows cannot be carried over. The operation renames the old table to
// PROJECT_LEGACY, with its primary key, and creates the target PROJECT beside it. The contract phase drops
// PROJECT_LEGACY, and refuses to while it holds rows.
//
// The previous release's project code does not work against the new table, so a rollback to the previous
// release cannot create or read projects. The old rows stay in PROJECT_LEGACY until contract.
func newReplaceProjectTableOperation() (engine.Operation, error) {
	return newSchemaChangeOperation(
		engine.OperationDefinition{
			ID:          replaceProjectTableOperationID,
			Description: "Keep the previous project table as PROJECT_LEGACY and create the target project table",
			StoreID:     engine.StoreIDEntityDB,
		},
		verifyProjectReplaced,
		[]engine.SQLStatement{
			{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
			{ID: "rename-project", Query: `ALTER TABLE "PROJECT" RENAME TO "PROJECT_LEGACY"`},
			// A renamed table keeps its primary key's name, PROJECT_pkey, which would leave the new table's key
			// to be named PROJECT_pkey1.
			{ID: "rename-project-legacy-primary-key", Query: `DO $$
DECLARE
    key_name TEXT;
BEGIN
    SELECT conname INTO key_name FROM pg_constraint
        WHERE conrelid = '"PROJECT_LEGACY"'::regclass AND contype = 'p';
    IF key_name IS NOT NULL AND key_name <> 'PROJECT_LEGACY_pkey' THEN
        EXECUTE format('ALTER TABLE "PROJECT_LEGACY" RENAME CONSTRAINT %I TO %I', key_name,
            'PROJECT_LEGACY_pkey');
    END IF;
END
$$`},
			{ID: "create-project", Query: postgresCreateProjectTableSQL},
			{ID: "create-project-handle-index", Query: `CREATE UNIQUE INDEX idx_project_handle
    ON "PROJECT" (DEPLOYMENT_ID, HANDLE)`},
		},
	)
}

// verifyProjectReplaced is at target when PROJECT has the target shape, at source when it still has the legacy
// shape and the names the operation is about to use are free, and invalid otherwise.
func verifyProjectReplaced(ctx context.Context, access engine.ReadAccess) (engine.VerificationResult, error) {
	relations, err := access.Query(ctx, engine.StoreIDEntityDB, queryTableRelation, projectTable.name)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect %s relation: %w", projectTable.name, err)
	}
	if len(relations) != 1 {
		return invalidVerification("unexpected_table_schema", fmt.Sprintf(
			"Expected one %s table, found %d", projectTable.name, len(relations))), nil
	}
	kind, err := relations[0].String("relation_kind")
	if err != nil {
		return engine.VerificationResult{}, err
	}
	if kind != "r" {
		return invalidVerification("unexpected_table_schema", fmt.Sprintf(
			"Relation name %s is not an ordinary table", projectTable.name)), nil
	}

	targetErr := validateTable(ctx, access, projectTable)
	if targetErr == nil {
		return engine.VerificationResult{State: engine.VerificationTarget,
			Summary: projectTable.name + " already matches the target schema"}, nil
	}

	rows, err := access.Query(ctx, engine.StoreIDEntityDB, queryTableColumns, projectTable.name)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect %s columns: %w", projectTable.name, err)
	}
	if !matchesColumns(rows, legacyProjectColumns) {
		return invalidVerification("unexpected_table_schema", fmt.Sprintf(
			"%s matches neither the previous release's schema nor the target schema: %v", projectTable.name,
			targetErr)), nil
	}
	for _, name := range []string{projectLegacyTable, projectTable.indexes[0].name} {
		taken, err := access.Query(ctx, engine.StoreIDEntityDB, queryTableRelation, name)
		if err != nil {
			return engine.VerificationResult{}, fmt.Errorf("inspect relation name %s: %w", name, err)
		}
		if len(taken) != 0 {
			return invalidVerification("relation_name_in_use", fmt.Sprintf(
				"Relation name %s is already in use, so %s cannot be replaced", name, projectTable.name)), nil
		}
	}
	return engine.VerificationResult{State: engine.VerificationSource,
		Summary: projectTable.name + " is ready to be kept as " + projectLegacyTable + " and replaced"}, nil
}

// matchesColumns reports whether the table's columns are exactly the expected ones, in order.
func matchesColumns(rows []engine.QueryRow, expected []columnSpec) bool {
	if len(rows) != len(expected) {
		return false
	}
	for index, want := range expected {
		actual, err := readColumn(rows[index])
		if err != nil || actual != want {
			return false
		}
	}
	return true
}

const postgresCreateProjectTableSQL = `CREATE TABLE "PROJECT" (
    DEPLOYMENT_ID   VARCHAR(255) NOT NULL,
    ID              VARCHAR(36)  NOT NULL,
    HANDLE          VARCHAR(100) NOT NULL,
    DISPLAY_NAME    VARCHAR(100) NOT NULL,
    DESCRIPTION     VARCHAR(255),
    CREATED_AT      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (DEPLOYMENT_ID, ID)
)`
