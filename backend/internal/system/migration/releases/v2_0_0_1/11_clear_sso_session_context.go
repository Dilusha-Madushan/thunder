// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"fmt"

	dbmodel "github.com/thunder-id/thunderid/internal/system/database/model"
	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

const (
	clearSSOSessionContextOperationID engine.OperationID = "2.0.0-m1-pidp/runtime_persistent/" +
		"11_clear_sso_session_context"

	// clearSSOSessionContextBatchSize is the number of context rows deleted per batch.
	clearSSOSessionContextBatchSize = 500
)

var (
	// The context table keeps its name in the target release. A row whose session is not in the new SESSION
	// table belongs to a session of the old binary, which the target cannot resume.
	queryDeleteOrphanSessionContext = dbmodel.DBQuery{
		ID: "MIG-USM-11-01",
		PostgresQuery: `DELETE FROM "SSO_SESSION_CONTEXT"
			WHERE (SESSION_ID, DEPLOYMENT_ID, CHECKPOINT_ID) IN (
				SELECT c.SESSION_ID, c.DEPLOYMENT_ID, c.CHECKPOINT_ID FROM "SSO_SESSION_CONTEXT" c
				WHERE ($1 = '' OR c.DEPLOYMENT_ID = $1)
					AND NOT EXISTS (SELECT 1 FROM "SESSION" s
						WHERE s.SESSION_ID = c.SESSION_ID AND s.DEPLOYMENT_ID = c.DEPLOYMENT_ID)
				LIMIT $2)`,
	}
	queryCountOrphanSessionContext = dbmodel.DBQuery{
		ID: "MIG-USM-11-02",
		PostgresQuery: `SELECT COUNT(*) AS orphan_count FROM "SSO_SESSION_CONTEXT" c
			WHERE ($1 = '' OR c.DEPLOYMENT_ID = $1)
				AND NOT EXISTS (SELECT 1 FROM "SESSION" s
					WHERE s.SESSION_ID = c.SESSION_ID AND s.DEPLOYMENT_ID = c.DEPLOYMENT_ID)`,
	}
)

// newClearSSOSessionContextOperation empties the session context the old binary left behind. It runs in
// migrate, after the old binary is retired, so no old session can write context again. Context rows of
// sessions the new binary already created are kept.
func newClearSSOSessionContextOperation() engine.Operation {
	return newBatchOperation(clearSSOSessionContextOperationID,
		"Delete the session context rows of sessions created by the previous release",
		engine.StoreIDRuntimePersistent, nil, readinessAfterSessionTable, verifySSOSessionContextCleared,
		clearSSOSessionContextBatch)
}

func clearSSOSessionContextBatch(
	ctx context.Context,
	access engine.ExecutionAccess,
	target engine.ExecutionTarget,
	checkpoint batchCheckpoint,
) (batchCheckpoint, bool, error) {
	deleted, err := access.Execute(ctx, engine.StoreIDRuntimePersistent, queryDeleteOrphanSessionContext,
		target.DeploymentID, clearSSOSessionContextBatchSize)
	if err != nil {
		return checkpoint, false, fmt.Errorf("delete session context rows: %w", err)
	}
	checkpoint.Processed += deleted
	return checkpoint, deleted < clearSSOSessionContextBatchSize, nil
}

func readinessAfterSessionTable(ctx context.Context, access engine.ReadAccess) (engine.ReadinessResult, error) {
	exists, err := relationExists(ctx, access, engine.StoreIDRuntimePersistent, sessionTable.name)
	if err != nil {
		return engine.ReadinessResult{}, err
	}
	if !exists {
		return engine.ReadinessResult{
			State: engine.ReadinessDeferred, Summary: "The operation waits for the SESSION table",
			DeferredUntil: []engine.OperationID{createSessionOperationID},
		}, nil
	}
	return readinessFromVerification(ctx, access, verifySSOSessionContextCleared)
}

func verifySSOSessionContextCleared(
	ctx context.Context,
	access engine.ReadAccess,
	deploymentID string,
) (engine.VerificationResult, error) {
	for _, table := range []string{sessionTable.name, "SSO_SESSION_CONTEXT"} {
		exists, err := relationExists(ctx, access, engine.StoreIDRuntimePersistent, table)
		if err != nil {
			return engine.VerificationResult{}, err
		}
		if !exists {
			return invalidVerification("missing_session_table", fmt.Sprintf("Table %s does not exist", table)), nil
		}
	}
	rows, err := access.Query(ctx, engine.StoreIDRuntimePersistent, queryCountOrphanSessionContext, deploymentID)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("count session context rows: %w", err)
	}
	if len(rows) != 1 {
		return engine.VerificationResult{}, fmt.Errorf("count session context rows: expected one row, found %d",
			len(rows))
	}
	count, err := rows[0].Int64("orphan_count")
	if err != nil {
		return engine.VerificationResult{}, err
	}
	if count == 0 {
		return engine.VerificationResult{
			State: engine.VerificationTarget, Summary: "No session context of the previous release remains",
		}, nil
	}
	return engine.VerificationResult{State: engine.VerificationSource, Summary: fmt.Sprintf(
		"%d session context rows of the previous release remain", count)}, nil
}
