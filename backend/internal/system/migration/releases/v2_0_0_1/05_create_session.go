// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import "github.com/thunder-id/thunderid/internal/system/migration/engine"

const createSessionOperationID engine.OperationID = "2.0.0-m1-pidp.1/runtime_persistent/05_create_session"

// sessionTable is the session table of the target release, which replaces SSO_SESSION. The old binary never
// refers to it, so it is created while the old binary serves.
var sessionTable = tableSpec{
	storeID: engine.StoreIDRuntimePersistent,
	name:    "SESSION",
	columns: []columnSpec{
		{"session_id", "NO", "character varying", "36", "none"},
		{"deployment_id", "NO", "character varying", "255", "none"},
		{"type", "NO", "character varying", "10", "none"},
		{"subject_id", "NO", "character varying", "36", "none"},
		{"flow_id", "NO", "character varying", "36", "none"},
		{"flow_version", "NO", "integer", "none", "none"},
		{"flow_execution_id", "NO", "character varying", "255", "none"},
		{"handle_id", "NO", "character varying", "255", "none"},
		{"authenticated_at", "NO", "timestamp without time zone", "none", "none"},
		{"created_at", "NO", "timestamp without time zone", "none", "current_timestamp"},
		{"last_active_at", "NO", "timestamp without time zone", "none", "none"},
		{"idle_expires_at", "YES", "timestamp without time zone", "none", "none"},
		{"absolute_expires_at", "YES", "timestamp without time zone", "none", "none"},
		{"state", "NO", "character varying", "50", "none"},
		{"version", "NO", "integer", "none", "none"},
		{"updated_at", "NO", "timestamp without time zone", "none", "current_timestamp"},
	},
	keys:   []keySpec{{typeCode: "p", columns: "session_id,deployment_id"}},
	checks: []checkSpec{{requiredFragments: []string{"type", "'SSO'", "'DIRECT'"}}},
	indexes: []indexSpec{
		{name: "idx_session_absolute_expires_at", unique: "false", columns: "absolute_expires_at"},
		{name: "idx_session_flow_execution", unique: "true", columns: "flow_execution_id,deployment_id"},
		{name: "idx_session_handle_id", unique: "true", columns: "handle_id,deployment_id"},
		{name: "idx_session_subject", unique: "false", columns: "deployment_id,subject_id"},
	},
}

func newCreateSessionOperation() (engine.Operation, error) {
	return newCreateTableOperation(
		createSessionOperationID,
		"Create the session table that replaces SSO_SESSION",
		sessionTable,
		[]engine.SQLStatement{
			{ID: "create-session-table", Query: postgresCreateSessionTableSQL},
			{ID: "create-session-handle-index", Query: `CREATE UNIQUE INDEX idx_session_handle_id
    ON "SESSION" (HANDLE_ID, DEPLOYMENT_ID)`},
			{ID: "create-session-flow-execution-index", Query: `CREATE UNIQUE INDEX idx_session_flow_execution
    ON "SESSION" (FLOW_EXECUTION_ID, DEPLOYMENT_ID)`},
			{ID: "create-session-absolute-expiry-index", Query: `CREATE INDEX idx_session_absolute_expires_at
    ON "SESSION" (ABSOLUTE_EXPIRES_AT)`},
			{ID: "create-session-subject-index", Query: `CREATE INDEX idx_session_subject
    ON "SESSION" (DEPLOYMENT_ID, SUBJECT_ID)`},
		},
	)
}

const postgresCreateSessionTableSQL = `CREATE TABLE "SESSION" (
    SESSION_ID VARCHAR(36) NOT NULL,
    DEPLOYMENT_ID VARCHAR(255) NOT NULL,
    TYPE VARCHAR(10) NOT NULL CHECK (TYPE IN ('SSO', 'DIRECT')),
    SUBJECT_ID VARCHAR(36) NOT NULL,
    FLOW_ID VARCHAR(36) NOT NULL,
    FLOW_VERSION INTEGER NOT NULL,
    FLOW_EXECUTION_ID VARCHAR(255) NOT NULL,
    HANDLE_ID VARCHAR(255) NOT NULL,
    AUTHENTICATED_AT TIMESTAMP NOT NULL,
    CREATED_AT TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    LAST_ACTIVE_AT TIMESTAMP NOT NULL,
    IDLE_EXPIRES_AT TIMESTAMP,
    ABSOLUTE_EXPIRES_AT TIMESTAMP,
    STATE VARCHAR(50) NOT NULL,
    VERSION INTEGER NOT NULL,
    UPDATED_AT TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (SESSION_ID, DEPLOYMENT_ID)
)`
