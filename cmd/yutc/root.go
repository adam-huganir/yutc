package main

import (
	"context"
	"fmt"
	"strings"

	yutc "github.com/adam-huganir/yutc/pkg"
	"github.com/adam-huganir/yutc/pkg/types"
	"github.com/adam-huganir/yutc/pkg/util"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type helpTopicFlag struct {
	set   bool
	topic string
}

func (h *helpTopicFlag) String() string {
	if h == nil {
		return "false"
	}
	if h.set {
		return "true"
	}
	return "false"
}

func (h *helpTopicFlag) Set(v string) error {
	h.set = true
	// When used as plain --help, pflag will set v to "true".
	// When used as --help=<topic>, v will be the topic.
	if strings.EqualFold(v, "true") {
		return nil
	}
	h.topic = strings.TrimSpace(v)
	return nil
}

func (h *helpTopicFlag) Type() string {
	// Must be "bool" to satisfy Cobra's internal help flag bool checks.
	return "bool"
}

func (h *helpTopicFlag) IsBoolFlag() bool {
	return true
}

func newRootCommand(settings *types.Arguments, runData *yutc.RunData, logger *zerolog.Logger) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "yutc [flags] <template_files...>",
		Short: "yutc - Yet Unnamed Templating CLI",
		Long:  `yutc is a command line tool for rendering complex templates from arbitrary sources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd.Context(), settings, runData, logger, args)
		},
		SilenceUsage: true,
	}
	return rootCommand
}

// ConfigureHelp sets up the custom help flags and usage printing with grouped flags.
func ConfigureHelp(cmd *cobra.Command, groups []*pflag.FlagSet) {
	// Ensure the default help flag exists, then swap its Value to a custom bool-compatible type
	// that can also capture an optional topic via --help=<topic>.
	cmd.InitDefaultHelpFlag()
	cmd.Flag("help").Usage = "Show help. A topic may be specified as --help=<topic>.\nAvailable topics:\n  syntax  Syntax for advanced file arguments and options"
	helpFlag := &helpTopicFlag{}
	if f := cmd.Flags().Lookup("help"); f != nil {
		f.Value = helpFlag
		f.DefValue = "false"
		f.NoOptDefVal = "true"
	}

	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		// Prefer explicit --help=<topic>, otherwise treat first remaining positional arg as topic.
		topicArg := strings.TrimSpace(helpFlag.topic)
		if topicArg == "" {
			remaining := c.Flags().Args()
			if len(remaining) > 0 {
				topicArg = remaining[0]
			}
		}
		if topicArg != "" {
			topic := strings.ToLower(strings.TrimSpace(topicArg))
			switch topic {
			case "syntax", "lexer":
				_, _ = fmt.Fprintln(c.OutOrStdout(), util.MustDedent(`
					Argument syntax help

					Some places files can be specified (the templates args and the flags --data/-d and --common-templates/-c) accept either:
					  1) A simple value:
					       ./my_file.yaml
					       https://example.com/my_file.yaml
					       -
					  2) A structured "key=value" format (comma-separated):
					       jsonpath=.Secrets,src=./my_secrets.yaml
					       src=./here.json,type=schema(defaults=false)

					Allowed keys:
					  src
					    The input source (file path, URL, or '-' for stdin).

					  jsonpath
					    Where to merge/nest the loaded data (ex: .Secrets becomes $.Secrets).
						Alternately, if a json schema is provided, this will specify where in the
						data to validate/resolve.

					  auth
					    URL auth in one of these forms:
					      username:password  (basic auth)
					      token              (bearer token)
					      "false"            (explicitly disable auth if a global auth is set)

					  type
					    Type modifier. Currently supports:
					      data
					      template
					      common
					      schema(defaults=true) # or false to disable defaults

					Notes:
					  - Field separator is ','
					  - To include a literal comma in a value, escape it as '\,'
					      src=my\,file.txt
					  - To include a literal ':' in auth, escape it as '\:'
					      auth=user\:password\,123

					Examples:
					  yutc -d ./values.yaml ./tmpl.tmpl
					  yutc -d jsonpath=.Secrets,src=./secrets.yaml ./tmpl.tmpl
					  yutc -d src=./schema.yaml,type=schema(defaults=false) ./tmpl.tmpl
					  yutc -d jsonpath=.Remote,src=https://example.com/data.yaml,auth=username:password ./tmpl.tmpl
				`))
				return
			default:
				_, _ = fmt.Fprintf(c.OutOrStdout(), util.MustDedent(`
					Unknown help topic: %s

					Available topics: syntax
				`), topicArg)
				return
			}
		}

		// Print standard usage header
		fmt.Fprintf(c.OutOrStdout(), "Usage:\n  %s\n\n", c.UseLine())
		// i don't like how this looks
		// if c.Short != "" {
		// fmt.Fprintf(c.OutOrStdout(), "%s\n", c.Short)
		// }
		if c.Long != "" {
			fmt.Fprintf(c.OutOrStdout(), "\n%s\n\n", c.Long)
		}

		// Print grouped flags using FlagSet.FlagUsages() which wraps natively
		for _, g := range groups {
			if g.HasFlags() {
				fmt.Fprintf(c.OutOrStdout(), "%s:\n%s\n", g.Name(), g.FlagUsages())
			}
		}
	})
}

func runRoot(ctx context.Context, settings *types.Arguments, runData *yutc.RunData, logger *zerolog.Logger, args []string) error {
	app := yutc.NewApp(settings, runData, logger)
	return app.Run(ctx, args)
}
