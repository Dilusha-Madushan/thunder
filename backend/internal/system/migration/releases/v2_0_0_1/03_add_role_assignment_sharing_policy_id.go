// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"errors"
	"fmt"

	dbmodel "github.com/thunder-id/thunderid/internal/system/database/model"
	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

const addRoleAssignmentSharingPolicyIDOperationID engine.OperationID = "2.0.0-m1-pidp/configdb/" +
	"03_add_role_assignment_sharing_policy_id"

var (
	queryRoleAssignmentTable = dbmodel.DBQuery{
		ID: "MIG-USM-03-01",
		PostgresQuery: `SELECT c.relkind::TEXT AS relation_kind
			FROM pg_class c
			INNER JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = CURRENT_SCHEMA() AND c.relname = 'ROLE_ASSIGNMENT'`,
	}
	queryRoleAssignmentSharingPolicyIDColumn = dbmodel.DBQuery{
		ID: "MIG-USM-03-02",
		PostgresQuery: `SELECT IS_NULLABLE AS is_nullable, DATA_TYPE AS data_type,
			COALESCE(CHARACTER_MAXIMUM_LENGTH::TEXT, 'none') AS maximum_length,
			COALESCE(COLUMN_DEFAULT, 'none') AS column_default
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = CURRENT_SCHEMA() AND TABLE_NAME = 'ROLE_ASSIGNMENT'
			AND COLUMN_NAME = 'sharing_policy_id'`,
	}
)

// newAddRoleAssignmentSharingPolicyIDOperation adds the column that records the sharing policy mandating a
// role assignment. The constant default is stored in the catalog, so the table is not rewritten. The old
// binary never names the column, and its inserts get the default, which marks a direct assignment.
func newAddRoleAssignmentSharingPolicyIDOperation() (engine.Operation, error) {
	return engine.NewSQLOperation(engine.SQLOperationConfig{
		Definition: engine.OperationDefinition{
			ID:               addRoleAssignmentSharingPolicyIDOperationID,
			Description:      "Add the sharing policy ID to role assignments",
			StoreID:          engine.StoreIDConfigDB,
			Kind:             engine.OperationKindSchemaChange,
			InterruptionMode: engine.InterruptionModeAtomic,
		},
		ReadinessChecker: checkRoleAssignmentSharingPolicyIDReadiness,
		PreVerifier:      verifyRoleAssignmentSharingPolicyID,
		Implementations: map[engine.DatabaseType]engine.SQLImplementation{
			engine.DatabaseTypePostgres: {
				Statements: []engine.SQLStatement{
					{ID: "set-lock-timeout", Query: "SET LOCAL lock_timeout = '2s'"},
					{ID: "add-sharing-policy-id", Query: `ALTER TABLE "ROLE_ASSIGNMENT"
    ADD COLUMN SHARING_POLICY_ID VARCHAR(36) NOT NULL DEFAULT ''`},
				},
			},
		},
		PostVerifier: verifyRoleAssignmentSharingPolicyID,
	})
}

func checkRoleAssignmentSharingPolicyIDReadiness(
	ctx context.Context,
	access engine.ReadAccess,
	_ engine.PlanContext,
) (engine.ReadinessResult, error) {
	verification, err := verifyRoleAssignmentSharingPolicyID(ctx, access, engine.ExecutionTarget{})
	if err != nil {
		return engine.ReadinessResult{}, err
	}
	if verification.State == engine.VerificationInvalid {
		return engine.ReadinessResult{
			State: engine.ReadinessBlocked, Summary: verification.Summary,
			Findings: []engine.Finding{{
				Code: "unexpected_role_assignment_schema", Message: verification.Summary,
				Remediation: "Restore the expected ROLE_ASSIGNMENT schema or reconcile the SHARING_POLICY_ID column",
			}},
		}, nil
	}
	return engine.ReadinessResult{State: engine.ReadinessReady, Summary: verification.Summary}, nil
}

func verifyRoleAssignmentSharingPolicyID(
	ctx context.Context,
	access engine.ReadAccess,
	_ engine.ExecutionTarget,
) (engine.VerificationResult, error) {
	tableRows, err := access.Query(ctx, engine.StoreIDConfigDB, queryRoleAssignmentTable)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect ROLE_ASSIGNMENT table: %w", err)
	}
	if len(tableRows) != 1 {
		return invalidRoleAssignmentSchema(fmt.Sprintf("Expected one ROLE_ASSIGNMENT table, found %d",
			len(tableRows))), nil
	}
	relationKind, err := tableRows[0].String("relation_kind")
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect ROLE_ASSIGNMENT relation kind: %w", err)
	}
	if relationKind != "r" {
		return invalidRoleAssignmentSchema("Relation name ROLE_ASSIGNMENT is not an ordinary table"), nil
	}

	columnRows, err := access.Query(ctx, engine.StoreIDConfigDB, queryRoleAssignmentSharingPolicyIDColumn)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect ROLE_ASSIGNMENT.SHARING_POLICY_ID column: %w", err)
	}
	if len(columnRows) == 0 {
		return engine.VerificationResult{
			State: engine.VerificationSource, Summary: "The ROLE_ASSIGNMENT.SHARING_POLICY_ID column is absent",
		}, nil
	}
	if len(columnRows) != 1 {
		return invalidRoleAssignmentSchema(fmt.Sprintf(
			"Expected at most one ROLE_ASSIGNMENT.SHARING_POLICY_ID column, found %d", len(columnRows))), nil
	}
	nullable, nullableErr := columnRows[0].String("is_nullable")
	dataType, dataTypeErr := columnRows[0].String("data_type")
	length, lengthErr := columnRows[0].String("maximum_length")
	defaultValue, defaultErr := columnRows[0].String("column_default")
	if err := errors.Join(nullableErr, dataTypeErr, lengthErr, defaultErr); err != nil {
		return engine.VerificationResult{}, err
	}
	if nullable != "NO" || dataType != "character varying" || length != "36" ||
		normalizeExpression(defaultValue) != "''::charactervarying" {
		return invalidRoleAssignmentSchema(fmt.Sprintf(
			"Unexpected ROLE_ASSIGNMENT.SHARING_POLICY_ID definition: nullable=%q, type=%q, length=%q, default=%q",
			nullable, dataType, length, defaultValue)), nil
	}
	return engine.VerificationResult{
		State:   engine.VerificationTarget,
		Summary: "The ROLE_ASSIGNMENT.SHARING_POLICY_ID column is present with an empty default",
	}, nil
}

func invalidRoleAssignmentSchema(summary string) engine.VerificationResult {
	return engine.VerificationResult{
		State: engine.VerificationInvalid, Summary: summary,
		Findings: []engine.Finding{{Code: "unexpected_role_assignment_schema", Message: summary}},
	}
}
