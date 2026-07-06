# k8s-model

Kubernetes sensor for the EmELand model. It watches EmELand CRDs and native cluster resources, feeds them into [modelsrv](https://github.com/emeland-io/modelsrv) `v0.9.3-rc3` as a library, and serves the modelsrv REST/OpenAPI API from the same process.

The operator embeds `go.emeland.io/modelsrv` (`backend.New()` + `endpoint.NewHandler`) rather than re-implementing the model store. K8s controllers convert cluster state into modelsrv domain types; the shared event pipeline (phase-0 findings, replication to subscribers) comes from the library.

## Release

Maintainers publish a release by pushing a semver git tag to `main`:

```bash
git tag v0.2.0
git push origin v0.2.0
```

CI then publishes:

- Operator image: `ghcr.io/emeland-io/modelsrv-k8s-sensor:0.2.0`
- CRD chart: `ghcr.io/emeland-io/charts/modelsrv-k8s-crd:0.2.0`
- Operator chart: `ghcr.io/emeland-io/charts/modelsrv-k8s-sensor:0.2.0`

Chart publish runs after the operator image workflow completes successfully.

### Install (sysadmin)

```bash
helm install modelsrv-k8s-crd oci://ghcr.io/emeland-io/charts/modelsrv-k8s-crd \
  --version 0.2.0 \
  --namespace emeland-system \
  --create-namespace

helm install modelsrv-k8s oci://ghcr.io/emeland-io/charts/modelsrv-k8s-sensor \
  --version 0.2.0 \
  --namespace emeland-system
```

See [charts/modelsrv-k8s-crd/README.md](charts/modelsrv-k8s-crd/README.md) and [charts/modelsrv-k8s-sensor/README.md](charts/modelsrv-k8s-sensor/README.md) for details.

## API

The sensor exposes the modelsrv REST API (default `:8080`, flag `--api-bind-address` / env `API_ADDR`):

- `GET /api/...` — read model resources
- `GET /swagger/` — Swagger UI
- `POST /api/events/register` — register downstream replication subscribers
- `GET /api/events/history`, `GET /api/events/subscribers` — event stream metadata

Inbound replication (`POST /api/events/push`) is **disabled by default** because cluster reconciliation is the source of truth. Use `--allow-inbound-push` to opt in.

Controller-runtime health probes remain on `:8081`; secure metrics on `:8443`.

## Authors

* [cypherfox](https://gitlab.com/cypherfox)

## License

This project is licensed under the Apache License 2.0.

You can find a copy of the license in the file [LICENSE](LICENSE)
