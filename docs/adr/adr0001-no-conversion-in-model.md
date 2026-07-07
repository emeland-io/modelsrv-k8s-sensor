# No Conversion in Model

## Status
active

## Context
Data structures have to be converted from K8s and OpenAPI formats into the internal format of the model.

## Decision
All transformations into the model structures happen in the K8s sensor adapter layer (`internal/controller`), not in `go.emeland.io/modelsrv/pkg/model`. The sensor uses modelsrv as a library; it does not maintain a forked model store.

## Consequences
* Controllers own K8s→domain conversion (`convertSystem`, `convertAPI`, etc.).
* The shared model, event pipeline, and REST API come from modelsrv `v0.9.3-rc3`.
* A name→UUID index in the sensor maps K8s resource names to modelsrv delete operations.
