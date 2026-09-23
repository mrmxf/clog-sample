//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package main

import (
	"testing"

	"github.com/mrmxf/util/check"
	"github.com/mrmxf/util/ci"
	"github.com/mrmxf/util/embedfs"
	"github.com/mrmxf/util/install"
	"github.com/mrmxf/util/kfg"
)

// A stack declares the tools CI must install before `clog build` runs. Those
// names are looked up in util/install's manifest, and nothing links the two
// packages - util/ci does not import util/install - so a stack can name a tool
// that cannot be installed and neither module's tests will notice.
//
// That has happened twice. The golang stacks declared [golang, trivy] while
// their own lint phase shell-outs to staticcheck and golangci-lint, so lint
// could not pass on a clean runner. And `trivy` itself was in no index at the
// time, so even the tools that WERE declared did not resolve.
//
// This binary is the only place both modules are linked, which makes it the
// only place the check can live.
func TestEveryStackToolCanBeInstalled(t *testing.T) {
	m, err := install.LoadManifest()
	if err != nil {
		t.Fatalf("cannot load the install manifest: %v", err)
	}

	for _, typ := range []string{
		ci.StackHugo, ci.StackGolang, ci.StackGolangLib, ci.StackContainer, ci.StackTinygo,
	} {
		stacks, err := ci.Stacks(ci.Config{
			Stack: ci.StackList{{Name: "probe", Type: typ}},
		})
		if err != nil {
			t.Errorf("stack type %q does not resolve: %v", typ, err)
			continue
		}
		tools := ci.StackTools(stacks)
		if len(tools) == 0 {
			t.Errorf("stack type %q declares no tools", typ)
			continue
		}
		for _, tool := range tools {
			if _, ok := m.Tools[tool]; !ok {
				t.Errorf("stack %q declares tool %q, which is not in util/install's index.yaml - "+
					"CI runs `clog Install %s` for this stack and it will fail",
					typ, tool, tool)
			}
		}
	}
}

// The check phases a stack names have to exist as check groups, or `clog build`
// resolves a phase list it cannot run. `clog BC flow` takes the names on trust.
func TestEveryStackChkPhaseHasACheckGroup(t *testing.T) {
	groups := checkGroups(t)

	for _, typ := range []string{
		ci.StackHugo, ci.StackGolang, ci.StackGolangLib, ci.StackContainer, ci.StackTinygo,
	} {
		stacks, err := ci.Stacks(ci.Config{
			Stack: ci.StackList{{Name: "probe", Type: typ}},
		})
		if err != nil {
			t.Errorf("stack type %q does not resolve: %v", typ, err)
			continue
		}
		for _, phase := range ci.StackChk(stacks) {
			if !groups[phase] {
				t.Errorf("stack %q names check phase %q, but no check.%s group is defined",
					typ, phase, phase)
			}
		}
	}
}

// checkGroups loads the shipped configuration and returns the check group
// names it defines. It deliberately loads util's embedded konfig WITHOUT the
// working-directory merge: the question is whether what clog ships is
// self-consistent, and a local .clog.yaml adding the missing group in this one
// repo would hide the gap from every other repo.
func checkGroups(t *testing.T) map[string]bool {
	t.Helper()
	opts := kfg.KonfigureOpt{
		AppFs:            embedfs.CoreFs,
		FilePath:         "konfig.yaml",
		PreventAutoMerge: true,
		PreventAutoApp:   true,
	}
	if err := kfg.Konfigure(&opts); err != nil {
		t.Fatalf("cannot load the embedded konfig: %v", err)
	}
	groups := map[string]bool{}
	for _, g := range check.Groups() {
		groups[g.Name] = true
	}
	if len(groups) == 0 {
		t.Fatal("the embedded konfig defines no check groups")
	}
	return groups
}
