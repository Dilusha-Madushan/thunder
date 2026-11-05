// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import "github.com/thunder-id/thunderid/internal/system/migration/engine"

const addOrganizationUnitProjectIDOperationID engine.OperationID = "2.0.0-m1-pidp/entitydb/" +
	"23_add_organization_unit_project_id"

// newAddOrganizationUnitProjectIDOperation adds the project an organization unit belongs to, and the index that
// lists organization units by project. The column is nullable with no default, like the entity column.
func newAddOrganizationUnitProjectIDOperation() (engine.Operation, error) {
	return newAddColumnOperation(addColumnSpec{
		id:          addOrganizationUnitProjectIDOperationID,
		description: "Add the project ID to organization units",
		storeID:     engine.StoreIDEntityDB,
		table:       "ORGANIZATION_UNIT",
		columns: []columnSpec{
			{"project_id", "YES", "character varying", "36", "none"},
		},
		indexes: []indexSpec{
			{name: "idx_ou_project", unique: "false", columns: "deployment_id,project_id"},
		},
		statements: []engine.SQLStatement{
			{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
			{ID: "add-project-id", Query: `ALTER TABLE "ORGANIZATION_UNIT" ADD COLUMN PROJECT_ID VARCHAR(36)`},
			{ID: "create-organization-unit-project-index", Query: `CREATE INDEX idx_ou_project
    ON "ORGANIZATION_UNIT" (DEPLOYMENT_ID, PROJECT_ID)`},
		},
	})
}
