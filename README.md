# Kubernetes DNS Manager

**Project status: *alpha*** 

Not all planned features are completed. 
The API, spec, status and other user facing objects are subject to change. 
We do not support backward-compatibility for the alpha releases.

## Overview

The DNS Operator allows managing DNS record directly from within a Kubernetes cluster by defining a new resource: DNSRecord.

When using an external provider, the DNS Operator will create a DNS record in the provider based on the resource definition.

When using the CoreDNS provider, the DNS Operator will configure CoreDNS to serve the DNS record.

The supported records types are:
- A
- CNAME
- TXT
- SRV
- MX

Example MX Record:
```yaml
apiVersion: dns.linka.cloud/v1alpha1
kind: DNSRecord
metadata:
  name: mx-example-org
  namespace: default
spec:
  mx:
    name: example.org.
    preference: 10
    target: mail.example.org.
```

### Raw DNS Records

**Only supported by the CoreDNS plugin**

For everything else the `raw` field allows to create any kind of record, including the supported ones.
Raw records are parsed using [miekg/dns](https://godoc.org/github.com/miekg/dns).

Example:
```yaml
apiVersion: dns.linka.cloud/v1alpha1
kind: DNSRecord
metadata:
  name: ns-example-org
  namespace: default
spec:
  raw: 'example.org ns ns0.dns.example.org'
```

### Generate A Records from LoadBalancer Services and Ingresses

The DNS Operator support creating automatically DNS records for LoadBalancer Services and Ingresses.

This behavior can be disabled by setting the `dns.linka.cloud/disabled` annotation on the Ingress or the Service.

The TTL can be set using the `dns.linka.cloud/ttl` annotation on the Ingress or the Service.

For Services, the DNS Operator will create an A record if the Service has the `dns.linka.cloud/hostname` annotation set 
to a valid dns hostname and the Service has a LoadBalancer IP.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: whoami
  annotations:
    dns.linka.cloud/hostname: whoami.example.org
    dns.linka.cloud/ttl: "60"
spec:
  selector:
    app: whoami
  ports:
  - port: 80
    name: http
  type: LoadBalancer
```

For Ingresses, the DNS Operator will create an A record per host with the status loadbalancer IP.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: whoami
  annotations:
    dns.linka.cloud/ttl: "60"
spec:
  rules:
  - host: whoami.example.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: whoami
            port:
              number: 80
```


## Requirements

### Domain Name

Obviously, you need a domain name to use this operator.

If you don't want to buy one, you can use a free domain name from [Freenom](https://www.freenom.com/en/index.html?lang=en)
and use it with [Cloudflare](https://www.cloudflare.com/) or the CoreDNS provider.

If using the CoreDNS provider, you will also need to configure your domain name to use the CoreDNS server as a nameserver.


### Cert-Manager

Cert Manager is required in order to generate the TLS certificates used by the DNS Operator Validation Webhook.

It can be installed using the [official documentation](https://cert-manager.io/docs/installation/kubernetes/).

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
```


## Installation

⚠️ **When upgrading from v0.1 to v0.2+, due to the renaming of the resource to the plural form, you need to back up all the DNSRecords then delete the old CRD before upgrading.** ⚠️

```bash

### CRDs, RBAC and Webhook

```bash
kubectl apply -f https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/common.yaml
```

### Providers

#### CoreDNS

*Note*:
In order to be available from outside the cluster, a LoadBalancer service is deployed with the operator.
The LoadBalancer external IP must be given to the operator by updating the deployment 
and setting the operator's `--external-address` flag.

Finally, change the nameservers in your DNS registrar console, so they point to the operator's 
coredns server.

```bash
kubectl apply -f https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/coredns.yaml
```

#### Cloudflare

Required environment variables:

- `CLOUDFLARE_TOKEN`: Cloudflare API token

```bash
export CLOUDFLARE_TOKEN=...
curl -sL https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/cloudflare.yaml | envsubst | kubectl apply -f -
```

#### Hetzner

Required environment variables:

- `HETZNER_TOKEN`: Hetzner DNS API token

```bash
export HETZNER_TOKEN=...
curl -sL https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/hetzner.yaml | envsubst | kubectl apply -f -
```

#### OVH

Required environment variables:

- `OVH_APPLICATION_KEY`: OVH Application Key
- `OVH_APPLICATION_SECRET`: OVH Application Secret
- `OVH_CONSUMER_KEY`: OVH Consumer Key
- `OVH_ENDPOINT`: OVH API endpoint

```bash
export OVH_APPLICATION_KEY=...
export OVH_APPLICATION_SECRET=...
export OVH_CONSUMER_KEY=...
export OVH_ENDPOINT=...
curl -sL https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/ovh.yaml | envsubst | kubectl apply -f -
```

#### Scaleway

Required environment variables:
- `SCALEWAY_SECRET_KEY`: Scaleway Secret Key
- `SCALEWAY_ORGANIZATION_ID`: Scaleway Organization ID

```bash
export SCALEWAY_SECRET_KEY=...
export SCALEWAY_ORGANIZATION_ID=...
curl -sL https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/scaleway.yaml | envsubst | kubectl apply -f -
```

## Uninstall

You need to delete the crds first, so that the controller can remove the finializers from the resources.
This will delete all the DNSRecords.

```bash
kubectl delete crds dnsrecords.dns.linka.cloud
```

Then delete the controller and the webhook.

```bash
kubectl delete -f https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/<provider>.yaml
kubectl delete -f https://raw.githubusercontent.com/linka-cloud/k8s-dns-manager/v0.2.0/deploy/common.yaml
```

## Operator

The operator ensure the dns records' validity and state (active / inactive).
When using the **coredns** provider, it may also run the CoreDNS server, but it should be soon moved out and deployed by the operator.
When using the other providers, the operator creates and updates the records using the DNS provider's API.

By default, the manifests include a Kubernetes LoadBalancer Service exposing the in-process CoreDNS server
`udp` and `tcp` ports: 53.

## k8s_dns CoreDNS Plugin
The `k8s_dns` plugin serve the `DNSRecord` and ensure that valid dns apex are served if not defined via `DNSRecord`:
- it generates a valid `NS` record for each dns records zones (e.g ns0.dns.example.org)
- it generates a valid `SOA` record for each dns records zones

In order to generate accurate `NS` records, the plugin needs to know the CoreDNS server public address.
It can be given using the `--external-address` operator's flag.

Next, the `NS` record should be configured in the DNS provider's console as Nameserver.

## Operator Configuration flags

```bash
$ k8s-dns --help

k8s-dns is a DNS Controller allowing to manage DNS Records from within a Kubernetes cluster

Usage:
  k8s-dns [flags]

Flags:
      --dns-any                      Enable coredns 'any' plugin
      --dns-cache int                Enable coredns cache with ttl (in seconds)
      --dns-forward strings          Dns forward servers
      --dns-log                      Enable coredns query logs
      --dns-metrics                  Enable coredns metrics on 0.0.0.0:9153
      --dns-verification-server ip   DNS server to use for verification (default 1.1.1.1)
      --enable-leader-election       Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
      --enable-webhook               Enable the validation webhook
  -a, --external-address ip          The external dns server address, e.g the loadbalancer service IP (default 127.0.0.1)
  -h, --help                         help for k8s-dns
      --metrics-addr string          The address the metric endpoint binds to. (default ":4299")
      --no-dns                       Do not run in process coredns server
  -p, --provider string              DNS provider to use (default "coredns")

```

## kubectl-dns

A `kubectl` plugin is available in the repository, it allows simple dns management tasks.

```bash
$ kubectl dns --help

dns root command

Usage:
  dns [command]

Available Commands:
  activate    active DNSRecord
  create      create a DNSRecord from bind record format and print it to stdout
  deactivate  de-activate DNSRecord
  help        Help about any command
  import      import dns bind file zone and print the DNSRecordList to stdout
  list        list DNSRecords

Flags:
  -h, --help   help for dns

Use "dns [command] --help" for more information about a command.

```

## Cert-Manager DNS Challenges Webhook

See [cert-manager-webhook-k8s-dns](https://github.com/linka-cloud/cert-manager-webhook-k8s-dns).

## Related Projects

- [Kubernetes ExternalDNS](https://github.com/kubernetes-sigs/external-dns)
- [Gardener External DNS Management](https://github.com/gardener/external-dns-management)

## TODOs:
- [ ] docs
- [ ] handle private IP address
- [ ] out of manager CoreDNS server
- [ ] CoreDNS server deployed by the manager
- [ ] find public CoreDNS server IP from LoadBalancer service
- [x] add CoreDNS options (cache, log, etc.)
