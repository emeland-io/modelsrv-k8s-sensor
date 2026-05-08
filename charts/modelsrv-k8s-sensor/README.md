# modelsrv-k8s-sensor

Deploys the EmELand Kubernetes sensor operator: `Deployment`, metrics `Service` (port 8443), RBAC, optional Gateway API `TLSRoute`, and optional `System` / `API` / `Component` CRs describing the operator.

## Prerequisites

- Kubernetes cluster.
- Install CRDs once (recommended separate release):

  ```bash
  helm install modelsrv-k8s-crd ./charts/modelsrv-k8s-crd --namespace emeland-system --create-namespace
  ```

- Optional: for `gateway.tlsRoute.enabled`, install the Gateway API **experimental** channel CRDs so `TLSRoute` exists.

## Install operator

Install the CRD chart first (see Prerequisites), then deploy the operator:

```bash
helm install modelsrv-k8s ./charts/modelsrv-k8s-sensor \
  --namespace emeland-system \
  --create-namespace \
  --set image.tag=<your-tag>
```

Default image repository: `registry.gitlab.com/emeland/k8s-model` (see `values.yaml`).

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

Bump chart `version` in `Chart.yaml` for chart changes; set `appVersion` to match the container image tag you deploy.
