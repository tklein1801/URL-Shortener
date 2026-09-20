```bash
fly -t ci set-pipeline \
  -p url-shortener-ci \
  -c .ci/pipelines/parent.yml \
  -l .ci/pipelines/vars.yml

fly -t ci validate-pipeline --strict \
  -c .ci/pipelines/parent.yml \
  -l .ci/pipelines/vars.yml
fly -t ci unpause-pipeline -p url-shortener-ci
```

The credential manager must provide `github.private_key`,
`github.status_token`, `github.release_token`, and `github.registry_token`. The
registry token must be allowed to read and write the GHCR package
`ghcr.io/tklein1801/url-shortener-server`; a classic personal access token needs
the `read:packages` and `write:packages` scopes. The parent pipeline creates and
archives branch pipeline instances automatically. It also checks open pull
requests targeting `main` and reports `concourse/pr-main`; the status token must
be able to read pull requests and repository contents and write commit statuses.

The worker used for `main-ci-release` must permit privileged tasks. Concourse
uses that capability only for the OCI image build; PR and branch pipelines do
not build or push container images.
