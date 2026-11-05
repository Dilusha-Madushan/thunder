// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import "github.com/thunder-id/thunderid/internal/system/migration/engine"

const createSessionParticipantOperationID engine.OperationID = "2.0.0-m1-pidp/runtime_persistent/" +
	"06_create_session_participant"

// sessionParticipantTable replaces SSO_SESSION_PARTICIPANT. OU_ID, the organization unit the application signed
// in under, is part of the primary key from the start, so no later key change is needed.
var sessionParticipantTable = tableSpec{
	storeID: engine.StoreIDRuntimePersistent,
	name:    "SESSION_PARTICIPANT",
	columns: []columnSpec{
		{"session_id", "NO", "character varying", "36", "none"},
		{"deployment_id", "NO", "character varying", "255", "none"},
		{"app_id", "NO", "character varying", "36", "none"},
		{"ou_id", "NO", "character varying", "36", "''::charactervarying"},
		{"tfid", "YES", "character varying", "36", "none"},
		{"first_joined_at", "NO", "timestamp without time zone", "none", "none"},
		{"last_active_at", "NO", "timestamp without time zone", "none", "none"},
	},
	keys: []keySpec{{typeCode: "p", columns: "session_id,deployment_id,app_id,ou_id"}},
	indexes: []indexSpec{
		{name: "idx_session_participant_app", unique: "false", columns: "deployment_id,app_id"},
	},
}

func newCreateSessionParticipantOperation() (engine.Operation, error) {
	return newCreateTableOperation(
		createSessionParticipantOperationID,
		"Create the session participant table that replaces SSO_SESSION_PARTICIPANT",
		sessionParticipantTable,
		[]engine.SQLStatement{
			{ID: "create-session-participant-table", Query: postgresCreateSessionParticipantTableSQL},
			{ID: "create-session-participant-app-index", Query: `CREATE INDEX idx_session_participant_app
    ON "SESSION_PARTICIPANT" (DEPLOYMENT_ID, APP_ID)`},
		},
	)
}

const postgresCreateSessionParticipantTableSQL = `CREATE TABLE "SESSION_PARTICIPANT" (
    SESSION_ID VARCHAR(36) NOT NULL,
    DEPLOYMENT_ID VARCHAR(255) NOT NULL,
    APP_ID VARCHAR(36) NOT NULL,
    OU_ID VARCHAR(36) NOT NULL DEFAULT '',
    TFID VARCHAR(36),
    FIRST_JOINED_AT TIMESTAMP NOT NULL,
    LAST_ACTIVE_AT TIMESTAMP NOT NULL,
    PRIMARY KEY (SESSION_ID, DEPLOYMENT_ID, APP_ID, OU_ID)
)`
