// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package v2_0_0_1

import (
	"context"
	"fmt"

	"github.com/thunder-id/thunderid/internal/system/migration/engine"
)

// addColumnSpec describes one column an operation adds to an existing table.
type addColumnSpec struct {
	id          engine.OperationID
	description string
	storeID     engine.StoreID
	table       string
	// column is the column as INFORMATION_SCHEMA reports it once added. Its position in the table is not
	// compared, because ALTER TABLE appends the column.
	column columnSpec
	// statements are the ordered statements that add it.
	statements []engine.SQLStatement
}

// newAddColumnOperation returns an atomic operation that adds one column to a table. The column is at source
// while absent and at target when it has exactly the expected type, nullability and default; any other
// column of that name, or a table that is not an ordinary table, is invalid.
func newAddColumnOperation(spec addColumnSpec) (engine.Operation, error) {
	verify := func(ctx context.Context, access engine.ReadAccess, _ engine.ExecutionTarget) (
		engine.VerificationResult, error) {
		return verifyAddedColumn(ctx, access, spec)
	}
	return engine.NewSQLOperation(engine.SQLOperationConfig{
		Definition: engine.OperationDefinition{
			ID: spec.id, Description: spec.description, StoreID: spec.storeID,
			Kind: engine.OperationKindSchemaChange, InterruptionMode: engine.InterruptionModeAtomic,
		},
		ReadinessChecker: func(ctx context.Context, access engine.ReadAccess, _ engine.PlanContext) (
			engine.ReadinessResult, error) {
			return readinessFromVerification(ctx, access,
				func(ctx context.Context, access engine.ReadAccess, _ string) (engine.VerificationResult, error) {
					return verifyAddedColumn(ctx, access, spec)
				})
		},
		PreVerifier: verify,
		Implementations: map[engine.DatabaseType]engine.SQLImplementation{
			engine.DatabaseTypePostgres: {Statements: spec.statements},
		},
		PostVerifier: verify,
	})
}

func verifyAddedColumn(ctx context.Context, access engine.ReadAccess, spec addColumnSpec) (
	engine.VerificationResult, error) {
	relations, err := access.Query(ctx, spec.storeID, queryTableRelation, spec.table)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect %s relation: %w", spec.table, err)
	}
	if len(relations) != 1 {
		return invalidVerification("unexpected_table_schema", fmt.Sprintf(
			"Expected one %s table, found %d", spec.table, len(relations))), nil
	}
	kind, err := relations[0].String("relation_kind")
	if err != nil {
		return engine.VerificationResult{}, err
	}
	if kind != "r" {
		return invalidVerification("unexpected_table_schema", fmt.Sprintf(
			"Relation name %s is not an ordinary table", spec.table)), nil
	}

	rows, err := access.Query(ctx, spec.storeID, queryTableColumns, spec.table)
	if err != nil {
		return engine.VerificationResult{}, fmt.Errorf("inspect %s columns: %w", spec.table, err)
	}
	for _, row := range rows {
		actual, err := readColumn(row)
		if err != nil {
			return engine.VerificationResult{}, err
		}
		if actual.name != spec.column.name {
			continue
		}
		if actual != spec.column {
			return invalidVerification("unexpected_column_definition", fmt.Sprintf(
				"Unexpected %s.%s definition: nullable=%q, type=%q, length=%q, default=%q", spec.table,
				spec.column.name, actual.nullable, actual.dataType, actual.maximumLength, actual.defaultValue)), nil
		}
		return engine.VerificationResult{State: engine.VerificationTarget,
			Summary: fmt.Sprintf("The %s.%s column is present", spec.table, spec.column.name)}, nil
	}
	return engine.VerificationResult{State: engine.VerificationSource,
		Summary: fmt.Sprintf("The %s.%s column is absent", spec.table, spec.column.name)}, nil
}
