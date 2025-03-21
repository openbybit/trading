package cmd

import (
	"log"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rand.Seed(time.Now().Unix())
}

var root = &cobra.Command{
	Use:   "bgw",
	Short: "bgw",
	Long:  `bgw`,
	Run: func(c *cobra.Command, args []string) {
		_ = c.Help()
	},
}

func Execute() {
	root.AddCommand(newServerCmd())

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
