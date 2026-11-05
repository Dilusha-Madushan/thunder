// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"fmt"

	dbmodel "github.com/thunder-id/thunderid/internal/system/database/model"
	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

const dropProjectLegacyTableOperationID engine.OperationID = "2.0.0-m1-pidp/entitydb/24_drop_project_legacy_table"

// queryProjectLegacyRowCount counts the projects of every deployment in the previous project table.
var queryProjectLegacyRowCount = dbmodel.DBQuery{
	ID:            "MIG-USM-24-01",
	PostgresQuery: `SELECT COUNT(*) AS row_count FROM "PROJECT_LEGACY"`,
}

// newDropProjectLegacyTableOperation drops the previous release's project table, which operation 21 kept as
// PROJECT_LEGACY. It does nothing when the table is not there.
//
// Its rows cannot be carried into the target table, so the drop is the step that would lose them. The operation
// is blocked while the table holds rows, and while PROJECT is not yet the target table, so nothing is dropped
// before its replacement is in place. The statements check for rows again inside the transaction, under a lock.
func newDropProjectLegacyTableOperation() (engine.Operation, error) {
	return newSchemaChangeOperation(
		engine.OperationDefinition{
			ID:          dropProjectLegacyTableOperationID,
			Description: "Drop the previous project table kept as PROJECT_LEGACY",
			StoreID:     engine.StoreIDEntityDB,
		},
		verifyProjectLegacyDropped,
		[]engine.SQLStatement{
			{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
			{ID: "require-empty-project-legacy", Query: `DO $$
BEGIN
    IF to_regclass('"PROJECT_LEGACY"') IS NOT NULL THEN
        LOCK TABLE "PROJECT_LEGACY" IN ACCESS EXCLUSIVE MODE;
        IF EXISTS (SELECT 1 FROM "PROJECT_LEGACY") THEN
            RAISE EXCEPTION 'PROJECT_LEGACY holds rows and cannot be dropped';
        END IF;
    END IF;
END
$$`},
			{ID: "drop-project-legacy", Query: `DROP TABLE IF EXISTS "PROJECT_LEGACY"`},
		},
	)
}

// verifyProjectLegacyDropped is at target when PROJECT_LEGACY is absent, at source when it is an empty table
// and PROJECT is the target table, and invalid when it holds rows, is not a table, or PROJECT is not yet the
// target table.
func verifyProjectLegacyDropped(ctx context.Context, access engine.ReadAccess) (engine.VerificationResult, error) {
	relations, err := access.Query(ctx, engine.StoreIDEntityDB, queryTableRelation, projectLegacyTable)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect %s relation: %w", projectLegacyTable, err)
	}
	if len(relations) == 0 {
		return engine.VerificationResult{State: engine.VerificationTarget,
			Summary: projectLegacyTable + " is absent"}, nil
	}
	kind, err := relations[0].String("relation_kind")
	if err != nil {
		return engine.VerificationResult{}, err
	}
	if kind != "r" {
		return invalidVerification("unexpected_table_schema", fmt.Sprintf(
			"Relation name %s is not an ordinary table", projectLegacyTable)), nil
	}

	replacement, err := verifyProjectReplaced(ctx, access)
	if err != nil {
		return engine.VerificationResult{}, err
	}
	if replacement.State != engine.VerificationTarget {
		return invalidVerification("project_not_replaced", fmt.Sprintf(
			"%s cannot be dropped before %s has the target schema: %s", projectLegacyTable, projectTable.name,
			replacement.Summary)), nil
	}

	count, err := access.Query(ctx, engine.StoreIDEntityDB, queryProjectLegacyRowCount)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("count projects: %w", err)
	}
	if len(count) != 1 {
		return engine.VerificationResult{}, fmt.Errorf("count projects: expected one row, found %d", len(count))
	}
	projects, err := count[0].Int64("row_count")
	if err != nil {
		return engine.VerificationResult{}, err
	}
	if projects > 0 {
		return invalidVerification("projects_not_migratable", fmt.Sprintf(
			"%s holds %d projects, and a project of the previous release has no handle or display name to carry "+
				"over. Copy them out, or add a migration for them, before dropping the table", projectLegacyTable,
			projects)), nil
	}
	return engine.VerificationResult{State: engine.VerificationSource,
		Summary: projectLegacyTable + " is empty and ready to be dropped"}, nil
}
