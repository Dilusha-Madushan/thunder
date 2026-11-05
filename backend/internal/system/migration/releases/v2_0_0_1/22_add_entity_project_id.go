// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import "github.com/thunder-id/thunderid/internal/system/migration/engine"

const addEntityProjectIDOperationID engine.OperationID = "2.0.0-m1-pidp/entitydb/22_add_entity_project_id"

// newAddEntityProjectIDOperation adds the project an entity belongs to, and the index that lists entities by
// project. The column is nullable with no default, so existing entities belong to no project and need no data
// migration. The index build blocks writes to ENTITY, which is acceptable because the migration runs while the
// server is stopped.
func newAddEntityProjectIDOperation() (engine.Operation, error) {
	return newAddColumnOperation(addColumnSpec{
		id:          addEntityProjectIDOperationID,
		description: "Add the project ID to entities",
		storeID:     engine.StoreIDEntityDB,
		table:       "ENTITY",
		columns: []columnSpec{
			{"project_id", "YES", "character varying", "36", "none"},
		},
		indexes: []indexSpec{
			{name: "idx_entity_project", unique: "false", columns: "deployment_id,category,project_id"},
		},
		statements: []engine.SQLStatement{
			{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
			{ID: "add-project-id", Query: `ALTER TABLE "ENTITY" ADD COLUMN PROJECT_ID VARCHAR(36)`},
			{ID: "create-entity-project-index", Query: `CREATE INDEX idx_entity_project
    ON "ENTITY" (DEPLOYMENT_ID, CATEGORY, PROJECT_ID)`},
		},
	})
}
