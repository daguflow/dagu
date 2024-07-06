package cmd

import (
	"log"

	"github.com/dagu-dev/dagu/internal/agent"
	"github.com/dagu-dev/dagu/internal/config"
	"github.com/dagu-dev/dagu/internal/dag"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [flags] <DAG file>",
		Short: "Runs the DAG",
		Long:  `dagu start [--params="param1 param2"] <DAG file>`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				// nolint
				log.Fatalf("Failed to load config: %v", err)
			}

			params, err := cmd.Flags().GetString("params")
			if err != nil {
				log.Fatalf("Failed to get params: %v", err)
			}

			dg, err := dag.Load(cfg.BaseConfig, args[0], params)
			if err != nil {
				log.Fatalf("Failed to load DAG: %v", err)
			}

			eng := newEngine(cfg)

			dagAgent := agent.New(&agent.NewAagentArgs{
				DAG: dg, LogDir: cfg.LogDir,
				Engine:    eng,
				DataStore: newDataStores(cfg),
			})

			ctx := cmd.Context()

			listenSignals(ctx, dagAgent)

			if err := dagAgent.Run(ctx); err != nil {
				log.Fatalf("Failed to start DAG: %v", err) // nolint // deep-exit
			}
		},
	}

	cmd.Flags().StringP("params", "p", "", "parameters")
	return cmd
}
