// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package api_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestSpecification(t *testing.T) {
	t.Parallel()

	spec, err := openapi3.NewLoader().LoadFromFile("api.yaml")
	require.NoError(t, err)
	require.NoError(t, spec.Validate(t.Context()))
}
