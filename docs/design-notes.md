# Design Notes

Some notes on why things they are, and the thoughts that went into the K8s encoding

## Desging Goals

* Encode as little as possible in the actual entities. Describe Systems through structure.
* Do not encode relationship in Spec of entity, unless required for composition. Create associations between entities of different technologies or domains of expertise through annotations instead.

## Identity

The manually created resources (System, API, Component)have UUIDs, as they need to be published between teams to create interoperability, as well as be referenced by their instances. But these UUIDs need only be assigned, when the entity is published or referenced. Therefore it is defined as optional in the basic entity.

Automatically created resources like the SystemInstance, ApiInstance and ComponentInstance get a UUID when they are created.

Names of Systems need only be local to the context they are created in. This requires a sufficiently powerful and stable context hierarchy.

An enforcement of uniqueness is required to ensure UUIDs are not used more than once.

## Findings

Findings are made available only through the OpenAPI interface. They are not created as K8s resources.

## Instantiation

### System 

* A system that is marked as abstract may not be referenced by a SystemInstance

### System Instance

* when a helm chart is instantiated as a new release, an annotation `systemInstanceId.emeland.io` with new UUID as value, MUST BE set on at least one `Deployment`, `DaemonSet`, or `StatefulSet`. Ideally all resources belonging to a system SHOULD be marked with this annotation. If more than one resource of a system instance is marked with this annotation, the UUID MUST BE the same for all resources.
