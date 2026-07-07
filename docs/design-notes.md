# Design Notes

Some notes on why things they are, and the thoughts that went into the K8s encoding

## Design Goals

* Encode as little as possible in the actual entities. Describe Systems through structure.
* Do not encode relationship in Spec of entity, unless required for composition. Create associations between entities of different technologies or domains of expertise through annotations instead.

## Architecture

The sensor is a thin wrapper around `go.emeland.io/modelsrv`:

* **K8s controllers** watch CRDs and native resources, convert them to modelsrv domain types (`system.NewSystem`, etc.), and call `model.Add*` / `Delete*ById`.
* **`backend.New()`** wires the shared model, event filter chain (phase-0 findings), and event manager.
* **`endpoint.NewHandler`** serves the REST/OpenAPI API and replication endpoints.

Conversion lives in `internal/controller` (see ADR 0001). The model store is **not** duplicated in this repo.

### Replication

The sensor is a **replication source**: downstream modelsrv instances can `POST /api/events/register` to receive pushes when cluster state changes. Inbound `POST /api/events/push` is rejected by default to avoid fighting the reconcile loop.

On restart, informers re-list and re-add resources (idempotent for subscribers), but deletions that occurred while the sensor was down are not replayed.

## Identity

The manually created resources (System, API, Component) have UUIDs, as they need to be published between teams to create interoperability, as well as be referenced by their instances. But these UUIDs need only be assigned, when the entity is published or referenced. Therefore it is defined as optional in the basic entity.

Automatically created resources like the SystemInstance, ApiInstance and ComponentInstance get a UUID when they are created.

Names of Systems need only be local to the context they are created in. This requires a sufficiently powerful and stable context hierarchy.

An enforcement of uniqueness is required to ensure UUIDs are not used more than once.

Controllers derive UUIDs from CRD spec fields when present, otherwise from `metadata.uid`. A name→UUID index maps K8s delete events to `Delete*ById` calls.

## Findings

Findings are made available only through the OpenAPI interface (via modelsrv's phase-0 filter chain). They are not created as K8s resources.

## Instantiation

### System

* A system that is marked as abstract may not be referenced by a SystemInstance

### System Instance

* when a helm chart is instantiated as a new release, an annotation `systemInstanceId.emeland.io` with new UUID as value, MUST BE set on at least one `Deployment`, `DaemonSet`, or `StatefulSet`. Ideally all resources belonging to a system SHOULD be marked with this annotation. If more than one resource of a system instance is marked with this annotation, the UUID MUST BE the same for all resources.
