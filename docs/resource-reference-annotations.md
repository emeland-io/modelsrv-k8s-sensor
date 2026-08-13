# Resource Reference Annotations

The k8s-sensor recognizes a set of Kubernetes annotations on native resources
to link them to EmELand model entities. When an annotation is present, the
sensor sets the corresponding first-class reference on the model object. When
the referenced resource does not exist locally, a finding is raised.

## Annotation Keys

| Annotation | K8s Resource | EmELand Entity | Reference Field |
|---|---|---|---|
| `emeland.io/k8s-sensor/api-reference` | Service, Ingress | ApiInstance | ApiRef |
| `emeland.io/k8s-sensor/component-reference` | Deployment, StatefulSet, DaemonSet, CronJob, Job | ComponentInstance | ComponentRef |
| `emeland.io/k8s-sensor/context-parent` | Namespace | Context | Parent |

All annotation values must be valid UUID strings (e.g.
`"550e8400-e29b-41d4-a716-446655440000"`).

### Legacy Keys

The following legacy annotation keys are still supported for backward
compatibility. If both the new and legacy key are present on the same resource,
the new key takes precedence.

| Legacy Key | Equivalent New Key |
|---|---|
| `apiId.emeland.io` | `emeland.io/k8s-sensor/api-reference` |
| `componentId.emeland.io` | `emeland.io/k8s-sensor/component-reference` |

## Behavior

### emeland.io/k8s-sensor/api-reference

Set this annotation on a K8s Service or Ingress to declare which EmELand API
the resulting ApiInstance represents.

- **Annotation absent:** The sensor emits a `MissingResourceReference` finding
  for the ApiInstance, indicating that the required reference is not configured.
- **Annotation present, UUID unknown locally:** The sensor emits a
  `ReferencedResourceNotFound` finding. The downstream `resolvefindings` filter
  in modelsrv will automatically delete this finding when the API resource
  appears later in the filter chain.
- **Annotation present, UUID known:** No finding is emitted. Any previously
  existing finding for this resource is cleared.

### emeland.io/k8s-sensor/component-reference

Set this annotation on a K8s Deployment, StatefulSet, DaemonSet, CronJob, or
Job to declare which EmELand Component the resulting ComponentInstance
implements.

Behavior is identical to `api-reference` above:

- Absent: `MissingResourceReference` finding.
- Present but unknown: `ReferencedResourceNotFound` finding.
- Present and known: No finding.

### emeland.io/k8s-sensor/context-parent

Set this annotation on a K8s Namespace to declare a non-cluster parent Context.
This is only needed for parent relationships beyond the default
namespace-to-cluster hierarchy (e.g. grouping namespaces under an application
context).

This annotation is **optional**. Its behavior differs from the other two:

- **Annotation absent:** No finding is emitted. The namespace still gets its
  implicit cluster parent (kube-system Context).
- **Annotation present, UUID unknown locally:** `ReferencedResourceNotFound`
  finding.
- **Annotation present, UUID known:** No finding. The Context's parent is set
  to the referenced Context (overriding the default cluster parent).

## Finding Types

The sensor registers two FindingType resources at startup:

| Kind | Stable UUID | Description |
|---|---|---|
| `ReferencedResourceNotFound` | `26a693f2-996d-5310-9e5b-a357722dcda5` | A resource references another resource by UUID that is not registered in the local model. |
| `MissingResourceReference` | `904c4012-fa93-5bbf-a8fe-7907eccce5d5` | A resource lacks a required EmELand reference to another resource. |

These UUIDs are derived deterministically from the kind string using UUID v5
(same namespace as modelsrv's `finding.TypeIDForKind`), so they match across
sensor and filter.

## Finding Resources Layout

Findings follow the subject-first convention expected by the modelsrv
`resolvefindings` filter:

- **ReferencedResourceNotFound:** `Resources = [subject, missing target]`
  - Example: `[ApiInstance(uid), API(referenced-uuid)]`
- **MissingResourceReference:** `Resources = [subject]`
  - Example: `[ApiInstance(uid)]`

## Finding Lifecycle

1. The sensor creates or updates the finding on every reconcile where the
   condition holds.
2. When the condition is resolved (annotation added or corrected, referenced
   resource appears locally), the sensor deletes its own finding.
3. For `ReferencedResourceNotFound`, the modelsrv `resolvefindings` filter
   provides a second resolution path: when the missing target resource arrives
   via a different sensor later in the filter chain, the filter deletes the
   finding even though the k8s-sensor has not re-reconciled.

## Example

```yaml
apiVersion: v1
kind: Service
metadata:
  name: payment-gateway
  namespace: production
  annotations:
    emeland.io/k8s-sensor/api-reference: "550e8400-e29b-41d4-a716-446655440000"
spec:
  selector:
    app: payment-gateway
  ports:
    - port: 443
      targetPort: 8443
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-gateway
  namespace: production
  annotations:
    emeland.io/k8s-sensor/component-reference: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: payment-gateway
  template:
    spec:
      containers:
        - name: gateway
          image: payment-gateway:1.2.3
---
apiVersion: v1
kind: Namespace
metadata:
  name: production
  annotations:
    emeland.io/k8s-sensor/context-parent: "f47ac10b-58cc-4372-a567-0e02b2c3d479"
```
