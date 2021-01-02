# Kubernetes DNS Operator

The DNS Operator define a new resource: DNSRecord.
This resource is then used the CoreDNS k8s_dns plugin to serve the records stored inside Kubernetes.

The supported records types are:
- A
- CNAME
- TXT
- SRV
- MX

For everything else the `raw` field allows to create any kind of record, including the supported ones.
Raw records are parsed using [miekg/dns](https://godoc.org/github.com/miekg/dns).

Example:
```yaml
apiVersion: dns.linka.cloud/v1alpha1
kind: DNSRecord
metadata:
  name: dns-google-com
  namespace: default
spec:
  raw: 'example.org ns ns0.dns.example.org'
```

## Operator

The operator only ensure (for now) the dns records' validity and state (active / inactive).
It also runs the CoreDNS server, but it should be soon moved out and deployed by the operator.

By default, the manifests include a Kubernetes LoadBalancer Service exposing the in-process CoreDNS server
`udp` port: 53.

## k8s_dns CoreDNS Plugin
The `k8s_dns` plugin serve the `DNSRecords` and ensure that valid dns apex are served if not defined via `DNSRecord`:
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
      --dns-cache int             Enable coredns cache with ttl (in seconds)
      --dns-forward strings       Dns forward servers
      --dns-log                   Enable coredns query logs
      --dns-metrics               Enable coredns metrics
      --enable-leader-election    Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
      --enable-webhook            Enable the validation webhook
  -a, --external-address string   The external dns server address, e.g the loadbalancer service IP (default "127.0.0.1")
  -h, --help                      help for k8s-dns
      --metrics-addr string       The address the metric endpoint binds to. (default ":8080")
      --no-dns                    Do not run in process coredns server
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


## TODOs:
- [ ] docs
- [ ] handle namespaces
- [ ] handle private IP address
- [ ] out of manager CoreDNS server
- [ ] CoreDNS server deployed by the manager
- [ ] find public CoreDNS server IP from LoadBalancer service
- [x] add CoreDNS options (cache, log, etc.)
