# clog-sample

**clog** is one vocabulary for building and deploying, on whatever runs your CI.

```
clog watch          the inner loop
clog build [dev|prod] [--fast]    every gate passed — ready to push
clog deploy [dev|prod]            publish what was built
```

Those three verbs mean the same thing on your laptop, in GitHub Actions and in
GitLab CI. The workflow file does not decide what they do — `.clog.yaml` does.

This repo is the worked example, and it is also the thing that ships: the
binaries other repos bootstrap from are built and released by running
`clog build prod && clog deploy prod` on this repo. **clog builds and deploys
itself with the workflow it is asking you to adopt.** If the vocabulary did not
hold, there would be no clog to download.

## The problem it removes

CI YAML traditionally carries the build. Which toolchain to install, which
branch may deploy, what "production" means — all of it inline, per workflow,
per engine, untestable, and different from what you run locally. Move to
another engine and you rewrite it; run it on your laptop and you can't.

clog inverts that. **Thin YAML, fat clog.** The workflow file says *when*; the
config says *what*; the binary does it. A whole GitHub build job is:

```yaml
jobs:
  build:
    uses: mrmxf/util/.github/workflows/build.yaml@workflows-v1
    with: {self-build: true}
```

The GitLab equivalent (`include: remote` of `util/gitlab/clog.gitlab-ci.yml`)
runs the same verbs against the same config. So does your terminal.

## What `.clog.yaml` says

Four keys carry a whole pipeline. This repo's, in full:

```yaml
ci:
  stack: golang                 # toolchain, checks and build steps, in one word
  policy:
    build: [branch, tag, pr, dispatch, local]
    deploy:
      prod: {tags: ["v*"], dispatch: true}
  targets:
    release:
      kind: github-release
      modes: [prod]
      prod: {repo: mrmxf/clog-sample, tag: "{tag}", dir: tmp,
             assets: clog-amd-lnx clog-arm-lnx clog-amd-mac clog-arm-mac}
```

- **`stack`** names what kind of repo this is — `hugo`, `golang`, `golang-lib`,
  `container`, `tinygo`. The type supplies the tools to install, the check
  phases (`pre-build lint test scan`) and the build phases. Override any of
  them per repo; most repos never do.
- **`policy`** decides whether a run builds and whether it deploys, from the
  event and the ref. A pull request never deploys, whatever it says.
- **`targets`** are deploy destinations, one block each. The kind picks the
  publisher; the `dev:`/`prod:` blocks carry its data, with `{tag}`,
  `{version}`, `{sha}` and `{mode}` expanded at deploy time.
- **`mode`** is `dev` or `prod`. There is no staging. `prod` refuses to build
  anything but a clean release tag, so a production binary is always a tag.

Ask the config what it decided, before pushing anything:

```console
$ clog ci policy      # will this run build? deploy? as what?
$ clog ci stack get chk
pre-build lint test scan
$ clog ci resolve     # what CI would check out, computed on your laptop
```

## Installing clog

`.clog-version` at your repo root pins the version; dev and CI install the
same bytes from it.

```yaml
- uses: mrmxf/util/.github/actions/setup-clog@workflows-v1
```

It downloads `clog-<cpu>-<os>` from this repo's releases and verifies it
against the release's `checksums.txt` before running it. On a laptop, the same
script does the same job:

```console
$ CLOG_VERSION_FILE=.clog-version ./get-clog.sh
```

## How this repo releases itself

```console
$ git tag v0.12.8 && git push origin v0.12.8
$ clog build prod          # checks, lints, scans → tmp/clog-<cpu>-<os>
$ clog deploy prod         # checksums those files, publishes the release
```

Pushing the tag runs exactly that in Actions — see
[.github/workflows/release.yaml](.github/workflows/release.yaml). The release
job downloads what the build job produced rather than rebuilding, so the bytes
that passed the gates are the bytes people download.

## More

- `clog ci --config-help` — every `.clog.yaml` key, including secrets via
  Infisical OIDC
- `clog build --help`, `clog ci policy --help`, `clog ci targets --help`
- [mrmxf/util](https://github.com/mrmxf/util) — the modules, the reusable
  workflows and the GitLab template
