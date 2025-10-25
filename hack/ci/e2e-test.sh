#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

# Install go-bindata
GO111MODULE=on go install github.com/go-bindata/go-bindata/...@latest

# Create kind cluster
kind create cluster --wait 5m

# Load the operator image into kind
kind load docker-image $IMAGE_NAME:$CI_COMMIT_REF_SLUG

# Install CRDs
make install

# Deploy operator
make deploy IMG=$IMAGE_NAME:$CI_COMMIT_REF_SLUG

# Run e2e tests
make test-e2e