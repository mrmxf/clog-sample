# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**clog-sample** is the public `clog` binary. It wires `github.com/mrmxf/util/*`
modules into a Cobra CLI and imports nothing private (no `utbd`, no
`clog-mrmxf`). Its releases are what `mrmxf/util`'s `setup-clog` action
downloads, so **every other repo's CI bootstraps from a release cut here**. A
broken release here breaks every consumer's pipeline, not just this repo.

The repo must therefore be able to build and release itself with the same verbs
it asks other repos to use. Keeping that true is the job.

## Build and Release Commands

### The verbs
- `clog build` — checks, lints, scans, then writes `tmp/clog-<cpu>-<os>`
- `clog build --fast` — skips every gate; makes no claim about the branch
- `clog build prod` — same, in prod mode; **refuses a HEAD that is not a clean
  release tag**, so a prod binary is always a tag
- `clog deploy prod` — checksums the `tmp/` assets and publishes the release
- Use `go run . <verb>` when iterating on this repo's own code

### Plain Go
- `go build -o clog .`, `go test ./...`, `go vet ./...`

### Cutting a release
1. Add the version to `releases.yaml` (**even** patch = `flow: main, build: prod`;
   odd = staging). Newest entry first.
2. Set `.clog-version` to the same tag — it is the pin both dev and CI install from.
3. Commit, `git push`, then `git tag vX.Y.Z && git push origin vX.Y.Z`.
4. `clog build prod && clog deploy prod`, or just let the tag push run
   `.github/workflows/release.yaml`.

The tag must be at `origin` before `clog deploy` runs: `bc-github-release`
checks for it and stops, because `gh release create` would otherwise invent a
release pointing at the wrong commit.

## Architecture

### Entry point
`main.go` embeds `releases.yaml` and layers config in an order that matters:

1. `util/embedfs` `konfig.yaml` — the base every clog app shares: the `bc-*`
   build workers, the four verbs, the `check:` groups. Without it the binary
   has no build tasks.
2. this repo's `.clog.yaml`.
3. the **working directory's** `.clog.yaml`, via a deferred `kfg.AutoMerge()`.

Step 3 is why `PreventAutoMerge: true` is set and `AutoMerge()` is called by
hand afterwards. When this binary runs in *another* repo's CI, that repo's
`.clog.yaml` must win. Let Konfigure do the merge and it fires first, and this
repo's demo config silently overrides the config of the repo being built.

`bootStrap()` registers the commands the embedded `bc-*` workers shell out to —
`BC`, `ci`, `Log`, `Crayon`, `Source`, `Check`, `Install`. Removing one does not
fail at startup; it fails part-way through someone else's build.

### `.clog.yaml`
The canonical and only repo config path. There is no `clogrc/clog.yaml` — that
path is legacy and is no longer searched.

- `ci.stack: golang` — one word that supplies the toolchain (`golang trivy`),
  the check phases (`pre-build lint test scan`) and the build phase (`golang`).
  Verify with `clog ci stack get chk` / `get tools` / `get make`.
- `ci.policy` — which events build, which refs deploy. `clog ci policy` prints
  the decision and its reason.
- `ci.targets.release` — `kind: github-release`, `modes: [prod]`. Asset names
  are **`clog-<cpu>-<os>`**, matching `util/.github/actions/setup-clog/get-clog.sh`.
- `snippets.project.config` sets `EXE=clog` (not `clog-sample`): `EXE` is the
  asset prefix, and setup-clog downloads `clog-<cpu>-<os>`. Built as
  `clog-sample-*` the release would be undownloadable. **Do not "fix" this.**
- `check.test` is named for the phase the golang stack asks for. Renaming it
  silently drops `go vet` and `go test` from the flow.
- There is deliberately no `check.tools` group: koanf replaces list blocks
  rather than merging them, so one here would shadow util's real one.

### `snippets.deploy` and `snippets.bc-github-release`
A **local override with a known expiry**. `util/ci/deploy.go`'s deployer table
implements `github-pages` only, so the base `deploy:` verb (`clog ci deploy`)
stops at *"kind github-release is valid but has no deployer yet"*. Until util
grows `deployGitHubRelease`, the publish lives in this repo's config.

`ci.targets.release` is already written in the shape that deployer will read.
When util ships one, **delete both snippets** — the base verb takes over
unchanged. Do not extend the override with logic that belongs in Go.

`bc-github-release` writes `checksums.txt` from inside the asset directory so
its lines carry bare filenames; `get-clog.sh` greps for exactly that form, and
refuses to install a binary it cannot verify. Assets and checksums are
published together or not at all.

## Workflows

- `.github/workflows/build.yaml` — pushes and PRs. Delegates to
  `mrmxf/util/.github/workflows/build.yaml@workflows-v1`.
- `.github/workflows/release.yaml` — `v*` tags and manual dispatch. Build job,
  then a release job with `contents: write` that **downloads the build job's
  artifact instead of rebuilding**, so the published bytes are the bytes that
  passed the gates. `GH_TOKEN: ${{ github.token }}` is what lets `gh` publish.

Both use `self-build: true`: this repo *is* clog, so `clog-prepare` compiles it
from this checkout rather than downloading a release. That is what breaks the
chicken-and-egg — never point these at `setup-clog`.

Keep the YAML thin. If CI needs new behaviour, it belongs in `.clog.yaml` or in
util, not in a `run:` step here; a step that exists only in CI is a step no
laptop can reproduce.

## Dependencies

`go.mod` pins published `github.com/mrmxf/util/*` module versions with no
`replace` directive. A change needed in util must be tagged and released there
before this repo can use it — plan work in that order.

## Gotchas

- **Do not add build logic to Go here.** This is a thin wiring `main.go`.
  Behaviour goes in `.clog.yaml` or upstream in util.
- `golangci-lint` and `trivy` must be on `PATH` for `clog build` to pass. clog
  refuses to install them behind your back — it names the tool and prints the
  install line (`clog Install <tool>`).
- `releases.yaml` is history, not policy. Versions come from git tags via
  `clog BC genBuildinfo`; nothing reads `releases.yaml` to decide what to build.
- `tmp/` is build output and is gitignored.
