# Concourse CI

The pipeline in this directory tests and builds both Go applications. Feature
and fix branches are exposed as Concourse pipeline instances; `main` is handled
by the parent pipeline because it also publishes the CLI release.

Every build runs a formatting check, `go vet`, and the complete race-enabled
test suite for both modules. It then builds the server for Linux amd64 and the
CLI for Linux, macOS, and Windows on amd64 and arm64. The build task uses Go
1.27.1, `CGO_ENABLED=0`, and `-trimpath`.

## GitHub status

Cogito posts the `concourse/ci` status for every branch and main build. It posts
`pending` after checkout and maps successful, failed, errored, and aborted jobs
to the corresponding GitHub state. The main release phase also uses the
`concourse/release` context.

The `pr-main` job detects open pull requests targeting `main`, checks the PR
merged with the current target branch, and runs the same tests and builds. The
pull-request resource reports this result as `concourse/pr-main` on the PR head
commit. This context can be configured as a required check in GitHub branch
protection.

## Releases

Each successful main commit creates or updates a GitHub prerelease. Its tag is
deterministic and has the form:

```text
main-YYYYMMDDTHHMMSSZ-<full-commit-sha>
```

The release contains six `surl` binaries and `SHA256SUMS`. Re-running a build
for the same commit updates the existing release and replaces its assets. The
release resource creates the release as a draft while assets are uploaded and
publishes it only after all uploads succeed.

## Installation

Copy `.ci/vars.example.yml` to an operator-managed variables file and provide
the three GitHub credentials through the configured Concourse credential
manager. The status token needs permission to read pull requests and repository
contents and to write commit statuses. The release token needs repository
contents write access. The SSH key must be able to read the repository.

```bash
fly -t <target> set-pipeline \
  -p url-shortener-ci \
  -c .ci/pipelines/parent.yml \
  -l .ci/pipelines/vars.yml

fly -t <target> unpause-pipeline -p url-shortener-ci
```

Validate the parent and template with representative values before setting the
pipeline. Keep real credentials in the credential manager.

The worker needs outbound access to the Go module proxy and must run Linux
amd64 containers. Full tests run as UID 65532 and require a writable `/tmp`.

PRs with merge conflicts fail during resource checkout, before a status hook can
use the checkout metadata. Changes made only to `main` do not retrigger every
open PR; push a new PR commit to run the merged verification again.
