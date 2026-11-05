// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"fmt"

	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

const dropSSOSessionTablesOperationID engine.OperationID = "2.0.0-m1-pidp/runtime_persistent/" +
	"17_drop_sso_session_tables"

// newDropSSOSessionTablesOperation drops the previous release's session and participant tables, which the
// target release replaces with SESSION and SESSION_PARTICIPANT. The new binary never uses them, so they are
// removed in contract with the rest of the obsolete representation; until then a rollback to the previous
// release still finds them. SSO_SESSION_CONTEXT keeps its name in the target release and is not dropped.
func newDropSSOSessionTablesOperation() (engine.Operation, error) {
	verify := func(ctx context.Context, access engine.ReadAccess, _ engine.ExecutionTarget) (
		engine.VerificationResult, error) {
		return verifySSOSessionTablesDropped(ctx, access)
	}
	return engine.NewSQLOperation(engine.SQLOperationConfig{
		Definition: engine.OperationDefinition{
			ID:               dropSSOSessionTablesOperationID,
			Description:      "Drop the session tables of the previous release",
			StoreID:          engine.StoreIDRuntimePersistent,
			Kind:             engine.OperationKindSchemaChange,
			InterruptionMode: engine.InterruptionModeAtomic,
		},
		ReadinessChecker: func(ctx context.Context, access engine.ReadAccess, _ engine.PlanContext) (
			engine.ReadinessResult, error) {
			return readinessFromVerification(ctx, access,
				func(ctx context.Context, access engine.ReadAccess, _ string) (engine.VerificationResult, error) {
					return verifySSOSessionTablesDropped(ctx, access)
				})
		},
		PreVerifier: verify,
		Implementations: map[engine.DatabaseType]engine.SQLImplementation{
			engine.DatabaseTypePostgres: {Statements: []engine.SQLStatement{
				{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
				{ID: "drop-sso-session-participant", Query: `DROP TABLE "SSO_SESSION_PARTICIPANT"`},
				{ID: "drop-sso-session", Query: `DROP TABLE "SSO_SESSION"`},
			}},
		},
		PostVerifier: verify,
	})
}

// verifySSOSessionTablesDropped is at source while both old tables exist and at target once both are gone.
// The drop does not depend on the old tables' exact shape, only on their presence.
func verifySSOSessionTablesDropped(ctx context.Context, access engine.ReadAccess) (
	engine.VerificationResult, error) {
	present := 0
	for _, table := range []string{"SSO_SESSION", "SSO_SESSION_PARTICIPANT"} {
		exists, err := relationExists(ctx, access, engine.StoreIDRuntimePersistent, table)
		if err != nil {
			return engine.VerificationResult{}, err
		}
		if exists {
			present++
		}
	}
	switch present {
	case 0:
		return engine.VerificationResult{State: engine.VerificationTarget,
			Summary: "The previous release's session tables are absent"}, nil
	case 2:
		return engine.VerificationResult{State: engine.VerificationSource,
			Summary: "The previous release's session tables are ready to be dropped"}, nil
	default:
		return invalidVerification("unexpected_sso_session_tables", fmt.Sprintf(
			"Only one of %s and %s exists", "SSO_SESSION", "SSO_SESSION_PARTICIPANT")), nil
	}
}
