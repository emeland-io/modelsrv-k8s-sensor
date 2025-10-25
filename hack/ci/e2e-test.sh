#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

echo 1. Install go-bindata
GO111MODULE=on go install github.com/go-bindata/go-bindata/...@latest

echo 2. Create kind cluster
kind create cluster --wait 5m

echo 3. Load the operator image into kind
kind load docker-image $IMAGE_NAME:$CI_COMMIT_REF_SLUG

echo 4. Install CRDs
make install

echo 5. Deploy operator
make deploy IMG=$IMAGE_NAME:$CI_COMMIT_REF_SLUG

echo 6. Run e2e tests
make test-e2e