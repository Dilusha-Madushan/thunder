// Copyright 2026 The ThunderID Authors
// SPDX-License-Identifier: Apache-2.0

package current

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinitionSelectsUserShareRelease(t *testing.T) {
	definition, err := Definition()

	require.NoError(t, err)
	assert.Equal(t, "2.0.0-m1-pidp.1", definition.Manifest.ReleaseVersion)
	assert.Equal(t, "2.0.0-m1-pidp.1", definition.Plan.TargetVersion)
}

func TestDefinitionForwardsDependenciesToCurrentRelease(t *testing.T) {
	definition, err := DefinitionWithDependencies(Dependencies{})

	require.NoError(t, err)
	assert.Equal(t, "2.0.0-m1-pidp.1", definition.Manifest.ReleaseVersion)
	assert.Equal(t, "2.0.0-m1-pidp.1", definition.Plan.TargetVersion)
}

func TestCatalogListsPreviousReleaseBeforeCurrentRelease(t *testing.T) {
	catalog, err := Catalog()

	require.NoError(t, err)
	require.Len(t, catalog, 2)
	assert.Equal(t, "2.0.0-m1-pidp", catalog[0].ReleaseVersion)
	assert.Equal(t, "2.0.0-m1-pidp.1", catalog[1].ReleaseVersion)
}
