// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

func TestDefinitionMatchesProductVersion(t *testing.T) {
	definition, err := Definition()
	require.NoError(t, err)

	versionBytes, err := os.ReadFile("../../../../../../version.txt")
	require.NoError(t, err)
	expectedVersion := strings.TrimPrefix(strings.TrimSpace(string(versionBytes)), "v")

	assert.Equal(t, expectedVersion, definition.Manifest.ReleaseVersion)
	assert.Equal(t, expectedVersion, definition.Plan.TargetVersion)
}

func TestDefinitionManifestAcceptsVersion2Stores(t *testing.T) {
	definition, err := Definition()
	require.NoError(t, err)

	require.Len(t, definition.Manifest.Stores, 4)
	for _, store := range definition.Manifest.Stores {
		assert.True(t, store.Changed, store.ID)
		assert.Equal(t, "2.0.0-m1-pidp.1", store.RequiredLevel, store.ID)
		if store.ID == engine.StoreIDRuntimeTransient {
			// The previous release left the runtime transient store at its first level.
			assert.Equal(t, "1.0.0", store.AcceptedSourceLevel, store.ID)
			continue
		}
		assert.Equal(t, "2.0.0-m1-pidp", store.AcceptedSourceLevel, store.ID)
	}
}

func TestDefinitionOrdersOperations(t *testing.T) {
	definition, err := Definition()
	require.NoError(t, err)

	phases := []struct {
		name       string
		operations []engine.Operation
		expected   []engine.OperationID
	}{
		{name: "expand", operations: definition.Plan.Expand, expected: []engine.OperationID{
			addInboundClientDeviceGrantFlowIDOperationID,
			createRuntimeStoreDeviceReqPartitionOperationID,
			addRoleAssignmentSharingPolicyIDOperationID,
			widenRoleAssignmentPrimaryKeyOperationID,
			createSessionOperationID,
			createSessionParticipantOperationID,
			createUserShareMigrationTableOperationID,
			snapshotDuplicateUsersOperationID,
			createOriginalUserSharePoliciesOperationID,
			addOriginalsToDuplicateGroupsOperationID,
		}},
		{name: "migrate", operations: definition.Plan.Migrate, expected: []engine.OperationID{
			clearSSOSessionContextOperationID,
			rehomeOriginalUsersOperationID,
			mergeDuplicateLinkedAccountsOperationID,
		}},
		{name: "contract", operations: definition.Plan.Contract, expected: []engine.OperationID{
			deleteDuplicateUserSessionsOperationID,
			deleteDuplicateUserAssignmentsOperationID,
			deleteDuplicateUsersOperationID,
			dropSSOSessionTablesOperationID,
			dropUserShareMigrationTableOperationID,
		}},
	}
	for _, phase := range phases {
		require.Len(t, phase.operations, len(phase.expected), phase.name)
		for index, expectedID := range phase.expected {
			assert.Equal(t, expectedID, phase.operations[index].Definition().ID, phase.name)
		}
	}

	expectedRollbacks := map[engine.OperationID]engine.OperationID{
		mergeDuplicateLinkedAccountsRollbackID:    mergeDuplicateLinkedAccountsOperationID,
		rehomeOriginalUsersRollbackID:             rehomeOriginalUsersOperationID,
		addOriginalsToDuplicateGroupsRollbackID:   addOriginalsToDuplicateGroupsOperationID,
		createOriginalUserSharePoliciesRollbackID: createOriginalUserSharePoliciesOperationID,
		snapshotDuplicateUsersRollbackID:          snapshotDuplicateUsersOperationID,
	}
	require.Len(t, definition.Plan.Rollback, len(expectedRollbacks))
	for _, rollback := range definition.Plan.Rollback {
		assert.Equal(t, expectedRollbacks[rollback.Definition().ID], rollback.ForwardOperationID())
	}
}

func TestDefinitionWithDependenciesMatchesDefinition(t *testing.T) {
	definition, err := DefinitionWithDependencies(Dependencies{})
	require.NoError(t, err)
	assert.Equal(t, "2.0.0-m1-pidp.1", definition.Manifest.ReleaseVersion)
}
