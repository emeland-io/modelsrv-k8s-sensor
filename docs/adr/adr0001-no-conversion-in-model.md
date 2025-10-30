# No Conversion in Model

## Status
active

## Context
Data structures have to be converted from K8s and OpenAPI formats into the internal format of the model.

## Decision
All transformations into the model structures have to happen outside the package `gitlab.com/emeland/k8s-model/internal/model`

## Consequences
