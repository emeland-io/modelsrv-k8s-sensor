# Setting up for End-to-End Testing

## Prerequisites

- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- [kind version 0.24+](https://kind.sigs.k8s.io/)
- [cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind)

## Set up Kubernetes cluster locally

1. Create Kind Cluster

```bash
kind create cluster
```

1. Ensure are nodes are allowed to run the external load balancer

```bash
kubectl label node kind-control-plane node.kubernetes.io/exclude-from-external-load-balancers-
```

1. Start external load-balancer

```bash
cloud-provider-kind >./cloud-provider-log &
```

Alternatively start the cloud-provider in a separate shell and keep that shell open.

## Configure Network

1. Set up self-signed Cert for traefik dashboard

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout tls.key -out tls.crt  \
            -subj "/CN=traefik.local"
kubectl create secret tls local-selfsigned-tls \
            --cert=tls.crt --key=tls.key \
            --namespace traefik
```

(TODO)

- Add entry for `traefik.local` and `modelsrv.local` to your local hosts file.

```bash
echo "$(INGRESS_EXTERNAL_IP) modelsrv.local traefik.local" >>/etc/hosts

## Cleanup

```bash
kind delete cluster
```
