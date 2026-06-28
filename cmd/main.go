package cmd

import (
	"fmt"
	"strings"
	"time"

	"reGit/dumper"

	"github.com/spf13/cobra"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	var branches []string
	var headers []string
	var userAgent string
	var proxy string
	var jobs int
	var retries int
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:          "reGit <url> <dir>",
		Short:        "Dump files from an exposed .git directory",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := strings.TrimSuffix(args[0], "/")
			outputDir := args[1]

			parsedHeaders, err := parseHeaders(headers)
			if err != nil {
				return err
			}

			handler, err := dumper.NewHandlerWithOptions(baseURL, outputDir, dumper.Options{
				Branches:  branches,
				Headers:   parsedHeaders,
				UserAgent: userAgent,
				Proxy:     proxy,
				Jobs:      jobs,
				Retries:   retries,
				Timeout:   timeout,
			})
			if err != nil {
				return err
			}

			return runWithProgress(handler)
		},
	}

	cmd.Flags().StringArrayVarP(&branches, "branch", "b", nil, "extra branch name to try")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "custom HTTP header, as 'Name: value'")
	cmd.Flags().StringVarP(&userAgent, "user-agent", "u", "", "custom HTTP user-agent")
	cmd.Flags().StringVar(&proxy, "proxy", "", "HTTP proxy URL")
	cmd.Flags().IntVarP(&jobs, "jobs", "j", 10, "number of concurrent file downloads")
	cmd.Flags().IntVarP(&retries, "retries", "r", 3, "retry count per request")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 3*time.Second, "HTTP request timeout")

	return cmd
}

func parseHeaders(values []string) (map[string]string, error) {
	headers := map[string]string{}
	for _, value := range values {
		name, headerValue, ok := strings.Cut(value, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("invalid header %q, expected 'Name: value'", value)
		}
		headers[strings.TrimSpace(name)] = strings.TrimSpace(headerValue)
	}
	return headers, nil
}
