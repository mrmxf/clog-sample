//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// Package main is the minimal public example of a clog-style CLI app.
// It wires together util/* modules only — no private utbd or clog-mrmxf imports.
package main

import (
	"embed"
	"log/slog"
	"os"
	"runtime"

	semver "github.com/mrmxf/util/buildinfo"
	"github.com/mrmxf/util/check"
	"github.com/mrmxf/util/install"
	"github.com/mrmxf/util/kfg"
	"github.com/mrmxf/util/snippets"
	"github.com/mrmxf/util/snips"
	"github.com/mrmxf/util/slogger"
	"github.com/spf13/cobra"
)

//go:embed releases.yaml
var ReleasesFs embed.FS

//go:embed clogrc/clog.yaml
var ClogYaml embed.FS

var debugFlag bool

var rootCmd = &cobra.Command{
	Use:   "clog-sample",
	Short: "A minimal clog sample app (util/* modules only)",
	Long: `clog-sample demonstrates the util/* modules working together as a CLI app.
It imports only github.com/mrmxf/util/* — no private utbd or clog-mrmxf packages.

Run without arguments to see this help. Use --debug for verbose logging.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func main() {
	defer slogger.CloseLogger()
	slogger.UsePrettyWithDbgTmpLogger(slog.LevelInfo)

	bootOpts := kfg.KonfigureOpt{
		AppFs:    ClogYaml,
		FilePath: "clogrc/clog.yaml",
		// disable merging from OS paths — sample is self-contained
		PreventAutoMerge: true,
		PreventAutoApp:   true,
	}
	if err := kfg.Konfigure(&bootOpts); err != nil {
		slog.Warn("config load failed", "err", err)
	}

	// set SemVer so commands that format version strings have data
	semver.Info()

	if err := bootStrap(rootCmd); err != nil {
		slog.Error("bootstrap failed", "err", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// bootStrap wires all util commands into the cobra tree.
func bootStrap(root *cobra.Command) error {
	// version / buildinfo
	root.AddCommand(semver.Command)

	// check: try/ok/catch/finally blocks from config
	root.AddCommand(check.Command)

	// declarative tool installer
	root.AddCommand(install.Command)

	// snippet tree from clogrc/clog.yaml
	snippetsKey := "snippets"
	rawSnippetsMap, _ := kfg.Unmarshal[snips.RawSnippets](snippetsKey, "")
	if rawSnippetsMap != nil {
		opts := snippets.Command{
			Use:     "Snippets",
			Key:     snippetsKey,
			Verbose: false,
			Plain:   false,
			Raw:     *rawSnippetsMap,
		}
		root.AddCommand(snippets.Bootstrap(root, opts))
	}

	return nil
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

	root := rootCmd
	root.PersistentFlags().BoolVar(&debugFlag, "debug", false, "enable debug logging")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if debugFlag {
			slogger.UsePrettyWithDbgTmpLogger(slog.LevelDebug)
			slog.Debug("debug logging enabled")
		}
	}
}
