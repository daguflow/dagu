// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package api_test

import (
	"net/http"
	"testing"

	"github.com/dagucloud/dagu/v2/api/v1"
	"github.com/dagucloud/dagu/v2/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSchedulerPauseServer(t *testing.T) test.Server {
	t.Helper()
	return setupWebhookTestServer(t)
}

func schedulerPauseState(t *testing.T, server test.Server, token string) api.SchedulerPauseState {
	t.Helper()
	resp := server.Client().Get("/api/v1/services/scheduler/pause").
		WithBearerToken(token).
		ExpectStatus(http.StatusOK).Send(t)
	var state api.SchedulerPauseState
	resp.Unmarshal(t, &state)
	return state
}

func TestSchedulerPause_DefaultsToRunning(t *testing.T) {
	t.Parallel()
	server := setupSchedulerPauseServer(t)
	adminToken := getWebhookAdminToken(t, server)

	state := schedulerPauseState(t, server, adminToken)

	assert.False(t, state.Paused)
	assert.Nil(t, state.PausedBy)
	assert.Nil(t, state.Reason)
}

func TestSchedulerPause_RoundTripRecordsActorAndReason(t *testing.T) {
	t.Parallel()
	server := setupSchedulerPauseServer(t)
	adminToken := getWebhookAdminToken(t, server)

	reason := "db migration"
	server.Client().Post("/api/v1/services/scheduler/pause", api.UpdateSchedulerPauseStateJSONRequestBody{
		Paused: true,
		Reason: &reason,
	}).WithBearerToken(adminToken).ExpectStatus(http.StatusOK).Send(t)

	state := schedulerPauseState(t, server, adminToken)
	require.True(t, state.Paused)
	require.NotNil(t, state.PausedBy)
	assert.Equal(t, "admin", *state.PausedBy)
	require.NotNil(t, state.Reason)
	assert.Equal(t, "db migration", *state.Reason)
	assert.NotNil(t, state.PausedAt)

	server.Client().Post("/api/v1/services/scheduler/pause", api.UpdateSchedulerPauseStateJSONRequestBody{
		Paused: false,
	}).WithBearerToken(adminToken).ExpectStatus(http.StatusOK).Send(t)

	resumed := schedulerPauseState(t, server, adminToken)
	assert.False(t, resumed.Paused)
	assert.Nil(t, resumed.PausedBy)
	assert.Nil(t, resumed.Reason)
}

// Pausing affects every workspace, so it is restricted to admins. Reading is
// deliberately open to every authenticated role, because a paused scheduler
// explains why nothing is running.
func TestSchedulerPause_MutationRequiresAdminButReadDoesNot(t *testing.T) {
	t.Parallel()
	server := setupSchedulerPauseServer(t)
	adminToken := getWebhookAdminToken(t, server)

	for _, u := range []struct {
		username string
		password string
		role     api.UserRole
	}{
		{"manager-user", "manager1", api.UserRoleManager},
		{"developer-user", "developer1", api.UserRoleDeveloper},
		{"operator-user", "operator1", api.UserRoleOperator},
		{"viewer-user", "viewerpass1", api.UserRoleViewer},
	} {
		server.Client().Post("/api/v1/users", api.CreateUserRequest{
			Username: u.username,
			Password: u.password,
			Role:     u.role,
		}).WithBearerToken(adminToken).ExpectStatus(http.StatusCreated).Send(t)

		resp := server.Client().Post("/api/v1/auth/login", api.LoginRequest{
			Username: u.username,
			Password: u.password,
		}).ExpectStatus(http.StatusOK).Send(t)
		var login api.LoginResponse
		resp.Unmarshal(t, &login)

		server.Client().Post("/api/v1/services/scheduler/pause", api.UpdateSchedulerPauseStateJSONRequestBody{
			Paused: true,
		}).WithBearerToken(login.Token).ExpectStatus(http.StatusForbidden).Send(t)

		state := schedulerPauseState(t, server, login.Token)
		assert.False(t, state.Paused, "%s must not have been able to pause", u.role)
	}
}

func TestSchedulerPause_ReadRequiresAuthentication(t *testing.T) {
	t.Parallel()
	server := setupSchedulerPauseServer(t)

	server.Client().Get("/api/v1/services/scheduler/pause").
		ExpectStatus(http.StatusUnauthorized).Send(t)
}
