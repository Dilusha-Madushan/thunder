// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

func TestClearSSOSessionContextBatch(t *testing.T) {
	access := newFakeAccess()
	access.affected[queryDeleteOrphanSessionContext.GetID()] = func([]interface{}) int64 {
		return clearSSOSessionContextBatchSize
	}

	checkpoint, completed, err := clearSSOSessionContextBatch(context.Background(), access,
		engine.ExecutionTarget{DeploymentID: "dep"}, batchCheckpoint{})

	require.NoError(t, err)
	assert.False(t, completed, "a full batch may leave more rows")
	assert.Equal(t, int64(clearSSOSessionContextBatchSize), checkpoint.Processed)
	assert.Equal(t, []interface{}{"dep", clearSSOSessionContextBatchSize}, access.executed[0].args)

	access.affected[queryDeleteOrphanSessionContext.GetID()] = func([]interface{}) int64 { return 7 }
	checkpoint, completed, err = clearSSOSessionContextBatch(context.Background(), access,
		engine.ExecutionTarget{DeploymentID: "dep"}, checkpoint)
	require.NoError(t, err)
	assert.True(t, completed)
	assert.Equal(t, int64(clearSSOSessionContextBatchSize+7), checkpoint.Processed)
}

func TestClearSSOSessionContextVerificationAndReadiness(t *testing.T) {
	operation := newClearSSOSessionContextOperation()
	definition := operation.Definition()
	assert.Equal(t, clearSSOSessionContextOperationID, definition.ID)
	assert.Equal(t, engine.StoreIDRuntimePersistent, definition.StoreID)

	orphans := func(count int64) *fakeAccess {
		return newFakeAccess().relations("SESSION", "SSO_SESSION_CONTEXT").
			on(queryCountOrphanSessionContext, engine.QueryRow{"orphan_count": count})
	}
	result, err := operation.PreVerify(context.Background(), orphans(3), engine.ExecutionTarget{})
	require.NoError(t, err)
	assert.Equal(t, engine.VerificationSource, result.State)
	result, err = operation.PostVerify(context.Background(), orphans(0), engine.ExecutionTarget{})
	require.NoError(t, err)
	assert.Equal(t, engine.VerificationTarget, result.State)

	readiness, err := operation.(engine.ReadinessChecker).CheckReadiness(context.Background(),
		newFakeAccess().relations("SSO_SESSION_CONTEXT"), engine.PlanContext{})
	require.NoError(t, err)
	assert.Equal(t, engine.ReadinessDeferred, readiness.State)
	assert.Equal(t, []engine.OperationID{createSessionOperationID}, readiness.DeferredUntil)
}

func TestDropSSOSessionTables(t *testing.T) {
	operation, err := newDropSSOSessionTablesOperation()
	require.NoError(t, err)
	definition := operation.Definition()
	assert.Equal(t, dropSSOSessionTablesOperationID, definition.ID)
	assert.Equal(t, engine.StoreIDRuntimePersistent, definition.StoreID)
	assert.Equal(t, engine.OperationKindSchemaChange, definition.Kind)

	implementation, err := operation.(*engine.SQLOperation).Implementation(engine.DatabaseTypePostgres)
	require.NoError(t, err)
	assert.Contains(t, implementation.Statements[1].Query, `DROP TABLE "SSO_SESSION_PARTICIPANT"`)
	assert.Contains(t, implementation.Statements[2].Query, `DROP TABLE "SSO_SESSION"`)

	tests := []struct {
		name   string
		tables []string
		state  engine.VerificationState
	}{
		{name: "tables present", tables: []string{"SSO_SESSION", "SSO_SESSION_PARTICIPANT"},
			state: engine.VerificationSource},
		{name: "tables dropped", state: engine.VerificationTarget},
		{name: "one table missing", tables: []string{"SSO_SESSION"}, state: engine.VerificationInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := operation.PreVerify(context.Background(), newFakeAccess().relations(test.tables...),
				engine.ExecutionTarget{})
			require.NoError(t, err)
			assert.Equal(t, test.state, result.State, result.Summary)
		})
	}
}
