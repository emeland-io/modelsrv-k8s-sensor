# modelsrv-k8s-sensor

Deploys the EmELand Kubernetes sensor operator: `Deployment`, metrics `Service` (port 8443), modelsrv REST API (port 8080), RBAC, optional Gateway API `TLSRoute`, and optional `System` / `API` / `Component` CRs describing the operator.

The operator embeds [modelsrv](https://github.com/emeland-io/modelsrv) `v0.9.3-rc3` and serves the REST/OpenAPI API on `manager.apiBindAddress` (default `:8080`, Service port `service.apiPort`).

## Prerequisites

- Kubernetes cluster.
- Install CRDs once (recommended separate release):

  ```bash
  helm install modelsrv-k8s-crd oci://ghcr.io/emeland-io/charts/modelsrv-k8s-crd \
    --version 0.2.0 \
    --namespace emeland-system \
    --create-namespace
  ```

- Optional: for `gateway.tlsRoute.enabled`, install the Gateway API **experimental** channel CRDs so `TLSRoute` exists.

## Install operator from GHCR (release)

Install the CRD chart first (see Prerequisites), then deploy the operator:

```bash
helm install modelsrv-k8s oci://ghcr.io/emeland-io/charts/modelsrv-k8s-sensor \
  --version 0.2.0 \
  --namespace emeland-system \
  --create-namespace
```

Replace `0.2.0` with the release version you want. The chart defaults to `ghcr.io/emeland-io/modelsrv-k8s-sensor:<Chart.AppVersion>` (no `--set image.tag` required for release installs).

## Install from source

Install the CRD chart first, then deploy the operator:

```bash
helm install modelsrv-k8s ./charts/modelsrv-k8s-sensor \
  --namespace emeland-system \
  --create-namespace \
  --set image.tag=<your-tag>
```

Default image repository: `ghcr.io/emeland-io/modelsrv-k8s-sensor` (see `values.yaml`). When `image.tag` is empty, the chart uses `Chart.AppVersion`.

### Optional: TLSRoute

Set `gateway.tlsRoute.enabled=true` and provide `gateway.tlsRoute.parentRefs` (and usually `hostnames`). Validate templates with:

```bash
helm template test ./charts/modelsrv-k8s-sensor \
  --set gateway.tlsRoute.enabled=true \
  --set-json 'gateway.tlsRoute.parentRefs=[{"group":"gateway.networking.k8s.io","kind":"Gateway","name":"gw","namespace":"gw-ns"}]' \
  --api-versions gateway.networking.k8s.io/v1alpha2
```

### Operator identity CRs

Set `operator.emelandIdentity.enabled=false` to skip creating `System`, `API`, and `Component` resources.

## Upgrading

For release installs, upgrade both charts to the same `--version`. After **`make manifests`**, Helm **`version`** and **`appVersion`** in both operator and CRD `Chart.yaml` files are rewritten from the Makefile **`VERSION`**.
