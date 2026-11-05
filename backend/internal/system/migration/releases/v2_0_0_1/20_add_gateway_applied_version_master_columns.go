// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import "github.com/thunder-id/thunderid/internal/system/migration/engine"

const addGatewayAppliedVersionMasterColumnsOperationID engine.OperationID = "2.0.0-m1-pidp/configdb/" +
	"20_add_gateway_applied_version_master_columns"

// newAddGatewayAppliedVersionMasterColumnsOperation adds the columns that record the master an apply left and
// whether the gateway holds the applied version whole. COMPLETE defaults to true, so a version applied by the
// previous release reads as applied whole; the constant default is stored in the catalog and the table is not
// rewritten. The other five columns are nullable with no default.
func newAddGatewayAppliedVersionMasterColumnsOperation() (engine.Operation, error) {
	return newAddColumnOperation(addColumnSpec{
		id:          addGatewayAppliedVersionMasterColumnsOperationID,
		description: "Add the master and completeness columns to gateway applied versions",
		storeID:     engine.StoreIDConfigDB,
		table:       "GATEWAY_APPLIED_VERSION",
		columns: []columnSpec{
			{"complete", "NO", "boolean", "none", "true"},
			{"master_seq", "YES", "integer", "none", "none"},
			{"master_hash", "YES", "character varying", "64", "none"},
			{"previous_master_seq", "YES", "integer", "none", "none"},
			{"resources_held", "YES", "integer", "none", "none"},
			{"resources_total", "YES", "integer", "none", "none"},
		},
		statements: []engine.SQLStatement{
			{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
			{ID: "add-master-columns", Query: `ALTER TABLE "GATEWAY_APPLIED_VERSION"
    ADD COLUMN COMPLETE BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN MASTER_SEQ INTEGER,
    ADD COLUMN MASTER_HASH VARCHAR(64),
    ADD COLUMN PREVIOUS_MASTER_SEQ INTEGER,
    ADD COLUMN RESOURCES_HELD INTEGER,
    ADD COLUMN RESOURCES_TOTAL INTEGER`},
		},
	})
}
