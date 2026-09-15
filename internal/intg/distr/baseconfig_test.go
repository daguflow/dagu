// Copyright (C) 2026 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package distr_test

import (
	"fmt"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dagucloud/dagu/v2/internal/cmn/config"
	"github.com/dagucloud/dagu/v2/internal/dispatch"
	"github.com/dagucloud/dagu/v2/internal/ir"
	"github.com/dagucloud/dagu/v2/internal/service/scheduler"
	"github.com/stretchr/testify/require"
)

func TestBaseConfig_SMTPRetry(t *testing.T) {
	for _, mode := range []string{"local", "direct", "queued"} {
		t.Run(mode, func(t *testing.T) {
			basePath := filepath.Join(t.TempDir(), "base.yaml")
			base, firstMail := receiveSMTP(t)
			require.NoError(t, os.WriteFile(basePath, []byte(base), 0600))
			selector := "worker_selector: {test: \"true\"}\n"
			if mode == "local" {
				selector = ""
			}
			f := newTestFixture(t, selector+`
name: smtp-retry
mail_on:
  failure: true
error_mail:
  from: sender@example.com
  to: recipient@example.com
  prefix: snapshot-smtp
steps:
  - name: fail
    run: exit 1
`, withBaseConfigPath(basePath), withWorkerBaseConfigPath(filepath.Join(t.TempDir(), "missing.yaml")))
			defer f.cleanup()
			if mode == "queued" {
				require.NoError(t, f.enqueue())
				f.waitForQueued()
				f.startScheduler(30 * time.Second)
			} else {
				require.NoError(t, f.start())
			}
			awaitSMTP(t, f, firstMail)
			status := f.waitForStatus(ir.Failed, 20*time.Second)
			f.waitForRunReleasedFromWorkers(status.DAGRunID, 10*time.Second)
			attempt, err := f.coord.DAGRunRepository.FindAttempt(f.coord.Context, status.DAGRun())
			require.NoError(t, err)
			snapshot, err := attempt.ReadDAG(f.coord.Context)
			require.NoError(t, err)

			latestBase, retryMail := receiveSMTP(t)
			require.NoError(t, os.WriteFile(basePath, []byte(latestBase), 0600))
			if mode == "local" {
				require.NoError(t, f.retry(status.DAGRunID))
			} else {
				executor := scheduler.NewDAGExecutor(f.coordinatorClient, nil, config.ExecutionModeDistributed, basePath)
				require.NoError(t, executor.ExecuteDAG(f.coord.Context, snapshot, dispatch.DispatchOperationRetry,
					status.DAGRunID, &status, ir.TriggerTypeRetry, ""))
			}
			awaitSMTP(t, f, retryMail)
			f.h.Wait.EventuallyEveryWithin("retry finishes in a new attempt", distrTestTimeout(20*time.Second), 50*time.Millisecond, func() bool {
				latest, err := f.latestStatus()
				return err == nil && latest.Status == ir.Failed && latest.AttemptID != status.AttemptID
			})
		})
	}
}

type smtpInbox struct {
	messages    <-chan string
	acknowledge chan struct{}
}

func awaitSMTP(t *testing.T, f *testFixture, inbox smtpInbox) {
	t.Helper()
	defer close(inbox.acknowledge)
	select {
	case message := <-inbox.messages:
		require.Contains(t, message, "snapshot-smtp")
		status, err := f.latestStatus()
		require.NoError(t, err)
		require.True(t, status.Status.IsActive(), "run must stay active until notification completes")
	case <-time.After(distrTestTimeout(20 * time.Second)):
		t.Fatal("SMTP notification was not received")
	}
}

// A separate listener for each attempt proves that retries use the current base.
func receiveSMTP(t *testing.T) (string, smtpInbox) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	messages := make(chan string, 1)
	acknowledge := make(chan struct{})
	done := make(chan struct{})
	t.Cleanup(func() { _ = listener.Close(); <-done })
	go func() {
		defer close(done)
		defer func() { _ = listener.Close() }()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(distrTestTimeout(20 * time.Second)))
		wire := textproto.NewConn(conn)
		if err := wire.PrintfLine("220 localhost"); err != nil {
			return
		}
		for {
			line, err := wire.ReadLine()
			if err != nil {
				return
			}
			switch {
			case line == "DATA":
				if err := wire.PrintfLine("354 Send message"); err != nil {
					return
				}
				body, err := wire.ReadDotBytes()
				if err != nil {
					return
				}
				messages <- string(body)
				select {
				case <-acknowledge:
				case <-t.Context().Done():
					return
				}
			case line == "QUIT":
				_ = wire.PrintfLine("221 Bye")
				return
			case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "HELO "),
				strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
			default:
				t.Errorf("unexpected SMTP command: %s", line)
				return
			}
			if err := wire.PrintfLine("250 OK"); err != nil {
				return
			}
		}
	}()
	return fmt.Sprintf("smtp:\n  host: %s\n  port: %q\n", host, port), smtpInbox{messages: messages, acknowledge: acknowledge}
}

func TestBaseConfig_EnvVarsExpandOnWorker(t *testing.T) {
	t.Run("baseConfigEnvVarsExpanded", func(t *testing.T) {
		// Create a base config with an env var
		baseDir := t.TempDir()
		baseConfigPath := filepath.Join(baseDir, "base.yaml")
		err := os.WriteFile(baseConfigPath, []byte(`env:
  - GITHUB_URL: "github.com"
`), 0600)
		require.NoError(t, err)

		f := newTestFixture(t, `
name: baseconfig-expand-test
worker_selector:
  test: "true"
steps:
  - name: use-base-env
    run: echo "${GITHUB_URL}"
`, withLogPersistence(), withBaseConfigPath(baseConfigPath))
		defer f.cleanup()

		require.NoError(t, f.enqueue())
		f.waitForQueued()
		f.startScheduler(30 * time.Second)

		status := f.waitForStatus(ir.Succeeded, 20*time.Second)
		require.Equal(t, ir.Succeeded, status.Status)
		assertLogContains(t, f.logDir(), f.dagWrapper.Name, status.DAGRunID, "use-base-env", "github.com")
	})
}

func TestBaseConfig_WorkerWithoutLocalBaseConfig(t *testing.T) {
	t.Run("workerUsesEmbeddedBaseConfig", func(t *testing.T) {
		// Create a base config only on the coordinator side
		baseDir := t.TempDir()
		baseConfigPath := filepath.Join(baseDir, "base.yaml")
		err := os.WriteFile(baseConfigPath, []byte(`env:
  - MY_SERVICE_URL: "https://api.example.com"
`), 0600)
		require.NoError(t, err)

		// The worker has a non-existent base config path.
		// This simulates k8s deployments where workers don't have local base configs.
		f := newTestFixture(t, `
name: no-local-base-test
worker_selector:
  test: "true"
steps:
  - name: use-embedded-env
    run: echo "${MY_SERVICE_URL}"
`, withLogPersistence(), withBaseConfigPath(baseConfigPath), withWorkerBaseConfigPath("/nonexistent/base.yaml"))
		defer f.cleanup()

		require.NoError(t, f.enqueue())
		f.waitForQueued()
		f.startScheduler(30 * time.Second)

		status := f.waitForStatus(ir.Succeeded, 20*time.Second)
		require.Equal(t, ir.Succeeded, status.Status)
		assertLogContains(t, f.logDir(), f.dagWrapper.Name, status.DAGRunID, "use-embedded-env", "https://api.example.com")
	})
}

func TestBaseConfig_MultipleEnvVarsMerged(t *testing.T) {
	t.Run("baseAndDAGEnvVarsMerged", func(t *testing.T) {
		baseDir := t.TempDir()
		baseConfigPath := filepath.Join(baseDir, "base.yaml")
		err := os.WriteFile(baseConfigPath, []byte(`env:
  - BASE_VAR1: "base-value-1"
  - BASE_VAR2: "base-value-2"
`), 0600)
		require.NoError(t, err)

		f := newTestFixture(t, `
name: merged-env-test
worker_selector:
  test: "true"
env:
  - DAG_VAR: "dag-value"
steps:
  - name: use-all-vars
    run: echo "${BASE_VAR1} ${BASE_VAR2} ${DAG_VAR}"
`, withLogPersistence(), withBaseConfigPath(baseConfigPath))
		defer f.cleanup()

		require.NoError(t, f.enqueue())
		f.waitForQueued()
		f.startScheduler(30 * time.Second)

		status := f.waitForStatus(ir.Succeeded, 20*time.Second)
		require.Equal(t, ir.Succeeded, status.Status)
		assertLogContains(t, f.logDir(), f.dagWrapper.Name, status.DAGRunID, "use-all-vars", "base-value-1 base-value-2 dag-value")
	})
}

func TestBaseConfig_SubDAGPropagation(t *testing.T) {
	t.Run("baseConfigForwardedToSubDAG", func(t *testing.T) {
		baseDir := t.TempDir()
		baseConfigPath := filepath.Join(baseDir, "base.yaml")
		err := os.WriteFile(baseConfigPath, []byte(`env:
  - BASE_VAR: "propagated-value"
`), 0600)
		require.NoError(t, err)

		f := newTestFixture(t, `
name: parent-with-base
steps:
  - name: call-child
    action: dag.run
    with:
      dag: child-dag

---
name: child-dag
worker_selector:
  type: test-worker
steps:
  - name: use-base-var
    run: echo "${BASE_VAR}"
`, withLogPersistence(), withBaseConfigPath(baseConfigPath), withLabels(map[string]string{"type": "test-worker"}))
		defer f.cleanup()

		// Use agent.RunSuccess() (direct execution) instead of the scheduler path
		// to test base config propagation through the sub-DAG dispatch specifically.
		agent := f.dagWrapper.Agent()
		agent.RunSuccess(t)
		f.dagWrapper.AssertLatestStatus(t, ir.Succeeded)
	})
}
