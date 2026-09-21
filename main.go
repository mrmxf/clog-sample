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

	"github.com/mrmxf/util/bc"
	semver "github.com/mrmxf/util/buildinfo"
	"github.com/mrmxf/util/check"
	"github.com/mrmxf/util/ci"
	"github.com/mrmxf/util/cmdlog"
	"github.com/mrmxf/util/crayon"
	"github.com/mrmxf/util/embedfs"
	"github.com/mrmxf/util/install"
	"github.com/mrmxf/util/kfg"
	"github.com/mrmxf/util/slogger"
	"github.com/mrmxf/util/snippets"
	"github.com/mrmxf/util/snips"
	"github.com/mrmxf/util/source"
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

	// Config is layered, and the order matters.
	//
	//   1. util's embedded konfig — the base every clog app shares. It carries
	//      the bc-* build workers (bc-hugo, bc-golang, bc-ko, bc-deploy-s3),
	//      the git:/project:/install: snippet trees and the check: blocks.
	//      Without it this binary has no build tasks to run.
	//   2. this app's own clogrc/clog.yaml — the sample's demo snippets.
	//   3. the working directory, via AutoMerge over kfg.try-paths, which ends
	//      with clogrc/clog.yaml and .clog.yaml.
	//
	// Step 3 is the whole point when another repo runs this binary in CI: that
	// repo's .clog.yaml is where its ci.policy/modes/targets live. AutoMerge is
	// deferred rather than left to Konfigure so it lands AFTER step 2 - run by
	// Konfigure it would fire first, and the sample's own demo config would then
	// override the config of the repo being built.
	bootOpts := kfg.KonfigureOpt{
		AppFs:            embedfs.CoreFs,
		FilePath:         "konfig.yaml",
		PreventAutoMerge: true,
		PreventAutoApp:   true,
	}
	if err := kfg.Konfigure(&bootOpts); err != nil {
		slog.Warn("base config load failed", "err", err)
	}
	if err := kfg.MergeKonfig(&kfg.KonfigureOpt{AppFs: ClogYaml, FilePath: "clogrc/clog.yaml"}); err != nil {
		slog.Debug("sample config merge failed", "err", err)
	}
	if err := kfg.AutoMerge(); err != nil {
		slog.Debug("working-directory config merge failed", "err", err)
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

	// ── build tasks ──────────────────────────────────────────────────────────
	// The embedded bc-* workers call all of these, so a binary without them can
	// load the build config and then fail part-way through running it:
	//   clog BC      flow, genBuildinfo, git, stashLog, is, semver   (96 calls)
	//   clog Log     worker logging                                  (24 calls)
	//   clog Crayon  terminal colours the workers eval               (10 calls)
	//   clog ci      resolve, policy, mode, targets, run, deploy      (7 calls)
	//   clog Source  `clog Source project config`                     (5 calls)
	root.AddCommand(bc.Command)
	root.AddCommand(ci.Command)
	root.AddCommand(cmdlog.Command)
	root.AddCommand(crayon.Command)
	root.AddCommand(source.Command)

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
