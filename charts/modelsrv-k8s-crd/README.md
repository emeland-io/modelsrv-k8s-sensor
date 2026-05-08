# modelsrv-k8s-crd

Installs the `structure.emeland.io` CRDs (`System`, `API`, `Component`, `SystemInstance`).

CRD files under `crds/` are produced from `config/crd/bases` (see `make copy-crd`; run after `make manifests` when APIs change).

## Install

```bash
helm install modelsrv-k8s-crd ./charts/modelsrv-k8s-crd
```

Install once per cluster. For upgrades, Helm applies CRD changes from the `crds/` directory when this chart is upgraded.

After CRDs are available, install the operator with [modelsrv-k8s-sensor](../modelsrv-k8s-sensor/README.md).
