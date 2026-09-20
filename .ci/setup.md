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
`github.status_token`, and `github.release_token`. The parent pipeline creates
and archives branch pipeline instances automatically.
