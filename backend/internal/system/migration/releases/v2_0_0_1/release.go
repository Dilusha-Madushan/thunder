// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

// Package v2_0_0_1 contains the migration that replaces per-organization duplicate users with one original
// user shared to the other organizations, and moves sessions to the renamed session tables.
package v2_0_0_1

import (
	_ "embed"

	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

//go:embed release-manifest.json
var manifestData []byte

// Dependencies contains the dependencies available while constructing this release's operations.
// Add narrowly typed providers only when a concrete operation requires them.
type Dependencies struct{}

// Definition returns the manifest and plan owned by this release.
func Definition() (*engine.ReleaseDefinition, error) {
	return DefinitionWithDependencies(Dependencies{})
}

// DefinitionWithDependencies returns the release definition constructed with the supplied dependencies.
func DefinitionWithDependencies(Dependencies) (*engine.ReleaseDefinition, error) {
	plan, err := migrationPlan()
	if err != nil {
		return nil, err
	}
	return engine.NewReleaseDefinition(manifestData, plan)
}

func migrationPlan() (engine.MigrationPlan, error) {
	addInboundClientDeviceGrantFlowID, err := newAddInboundClientDeviceGrantFlowIDOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	createRuntimeStoreDeviceReqPartition, err := newCreateRuntimeStoreDeviceReqPartitionOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	addRoleAssignmentSharingPolicyID, err := newAddRoleAssignmentSharingPolicyIDOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	createSession, err := newCreateSessionOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	createSessionParticipant, err := newCreateSessionParticipantOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	createUserShareMigrationTable, err := newCreateUserShareMigrationTableOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	dropSSOSessionTables, err := newDropSSOSessionTablesOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	dropUserShareMigrationTable, err := newDropUserShareMigrationTableOperation()
	if err != nil {
		return engine.MigrationPlan{}, err
	}
	return engine.MigrationPlan{
		TargetVersion: "2.0.0-m1-pidp.1",
		// Expand runs while the previous release still serves, in login-only mode. Everything here is
		// either invisible to it or harmless to it.
		Expand: []engine.Operation{
			addInboundClientDeviceGrantFlowID,
			createRuntimeStoreDeviceReqPartition,
			addRoleAssignmentSharingPolicyID,
			newWidenRoleAssignmentPrimaryKeyOperation(),
			createSession,
			createSessionParticipant,
			createUserShareMigrationTable,
			newSnapshotDuplicateUsersOperation(),
			newCreateOriginalUserSharePoliciesOperation(),
			newAddOriginalsToDuplicateGroupsOperation(),
		},
		// Migrate runs in the new binary after the previous release is retired.
		Migrate: []engine.Operation{
			newClearSSOSessionContextOperation(),
			newRehomeOriginalUsersOperation(),
			newMergeDuplicateLinkedAccountsOperation(),
		},
		// Contract removes what the new binary no longer uses: the duplicate users, the previous release's
		// session tables, and last the working table.
		Contract: []engine.Operation{
			newDeleteDuplicateUserSessionsOperation(),
			newDeleteDuplicateUserAssignmentsOperation(),
			newDeleteDuplicateUsersOperation(),
			dropSSOSessionTables,
			dropUserShareMigrationTable,
		},
		Rollback: []engine.RollbackOperation{
			newRollbackMergeDuplicateLinkedAccountsOperation(),
			newRollbackRehomeOriginalUsersOperation(),
			newRollbackAddOriginalsToDuplicateGroupsOperation(),
			newRollbackCreateOriginalUserSharePoliciesOperation(),
			newRollbackSnapshotDuplicateUsersOperation(),
		},
	}, nil
}
