package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	_ "go.uber.org/automaxprocs"

	"bgw/pkg/server"
)

func newServerCmd() *cobra.Command {
	shardIndex := -1

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "serve",
		Long:  `serve (http|ws|grpc|all)`,
		PreRun: func(cmd *cobra.Command, args []string) {
			log.Println("start server")
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				os.Exit(1)
			}

			server.Run(args[0], shardIndex)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			log.Println("server stopped")
		},
	}

	cmd.Flags().IntVarP(&shardIndex, "port_index", "p", -1, "port index, default -1")

	return cmd
}
