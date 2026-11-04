// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

// Package current explicitly selects the migration definition compiled into this binary.
package current

import (
	"github.com/thunder-id/thunderid/internal/system/migration/engine"
	previous "github.com/thunder-id/thunderid/internal/system/migration/releases/v2_0_0"
	release "github.com/thunder-id/thunderid/internal/system/migration/releases/v2_0_0_1"
)

// Dependencies contains the dependencies available to the release compiled into this binary.
type Dependencies = release.Dependencies

// Definition returns the release definition selected for the current binary.
func Definition() (*engine.ReleaseDefinition, error) {
	return DefinitionWithDependencies(Dependencies{})
}

// DefinitionWithDependencies returns the current release definition with the supplied dependencies.
func DefinitionWithDependencies(dependencies Dependencies) (*engine.ReleaseDefinition, error) {
	return release.DefinitionWithDependencies(dependencies)
}

// Catalog returns the ordered migration manifests known to the current binary. The previous release is
// listed so that a store still at its source level is routed through it first.
func Catalog() ([]engine.ReleaseManifest, error) {
	previousDefinition, err := previous.Definition()
	if err != nil {
		return nil, err
	}
	definition, err := release.Definition()
	if err != nil {
		return nil, err
	}
	return []engine.ReleaseManifest{previousDefinition.Manifest, definition.Manifest}, nil
}
