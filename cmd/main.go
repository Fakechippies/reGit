package cmd

import (
	"strings"

	"reGit/dumper"

	"github.com/spf13/cobra"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reGit <url> <dir>",
		Short: "Dump files from an exposed .git directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := strings.TrimSuffix(args[0], "/")
			outputDir := args[1]

			handler, err := dumper.NewHandler(baseURL, outputDir)
			if err != nil {
				return err
			}

			return handler.Run()
		},
	}

	return cmd
}
