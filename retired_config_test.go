//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mrmxf/util/embedfs"
)

// retiredSpellings are the command paths v1.0.0 renamed. A retired command
// fails loudly when it runs, which is what makes the rename safe in a script
// somebody wrote last year - but config that SHIPS should never contain one,
// because that failure would be clog's own fault rather than the caller's.
//
// This turns those runtime errors into a build failure for the config we
// control: util's embedded konfig and this repo's .clog.yaml.
var retiredSpellings = []struct{ pattern, use string }{
	{`clog\s+BC\s+stashLog\b`, "clog BC stash log"},
	{`clog\s+BC\s+genBuildinfo\b`, "clog BC gen buildinfo"},
	{`clog\s+BC\s+linkerpath\b`, "clog BC get linkerpath"},
	{`clog\s+BC\s+is\s`, "clog BC releases is"},
	{`clog\s+BC\s+semver\s+"`, "clog BC semver satisfies"},
	{`clog\s+BC\s+git\s+(branch|suffix)\b`, "clog BC git get <branch|suffix>"},
	{`clog\s+BC\s+git\s+tag\s+(head|origin|prod|production|ref)\b`, "clog BC git tag get <which>"},
	{`clog\s+BC\s+git\s+hash\s+(head|origin|prod|production|ref)\b`, "clog BC git hash get <which>"},
	{`clog\s+BC\s+git\s+tree\s+(clean|ahead|behind|unstaged)\b`, "clog BC git tree is|has <state>"},
	{`clog\s+BC\s+releases\s+(version|date|flow|note|build|yaml)\b`, "clog BC releases get <field>"},
	{`clog\s+CI\s+policy\b`, "clog CI show policy"},
	{`clog\s+CI\s+targets\b`, "clog CI target list"},
	{`clog\s+CI\s+resolve\b`, "clog CI show event"},
	{`clog\s+CI\s+stack\s+get\s+(tools|chk|make|names)\b`, "clog CI stack list <which>"},
	{`clog\s+CI\s+scan\s+--list-targets\b`, "clog CI scan list targets"},
}

func TestShippedConfigUsesNoRetiredSpelling(t *testing.T) {
	type source struct {
		name string
		body string
	}
	var sources []source

	// util's embedded konfig - what every clog app inherits
	err := fs.WalkDir(embedfs.CoreFs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return err
		}
		b, rerr := fs.ReadFile(embedfs.CoreFs, p)
		if rerr != nil {
			return rerr
		}
		sources = append(sources, source{"util embedfs:" + p, string(b)})
		return nil
	})
	if err != nil {
		t.Fatalf("cannot walk the embedded konfig: %v", err)
	}

	// this repo's own config
	for _, f := range []string{".clog.yaml"} {
		b, rerr := os.ReadFile(filepath.Clean(f))
		if rerr != nil {
			t.Fatalf("cannot read %s: %v", f, rerr)
		}
		sources = append(sources, source{f, string(b)})
	}
	if len(sources) < 2 {
		t.Fatal("no config sources found - this test would pass vacuously")
	}

	for _, r := range retiredSpellings {
		re := regexp.MustCompile(r.pattern)
		for _, src := range sources {
			for i, line := range strings.Split(src.body, "\n") {
				// a line that documents the rename is not a call site
				if strings.Contains(line, "retired") || strings.Contains(line, "->") {
					continue
				}
				if re.MatchString(line) {
					t.Errorf("%s:%d uses a retired spelling; use `%s`\n  %s",
						src.name, i+1, r.use, strings.TrimSpace(line))
				}
			}
		}
	}
}
