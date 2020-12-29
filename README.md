# Kubernetes DNS Operator

The DNS Operator define a new resource: DNSRecord.
This resource is then used the CoreDNS k8s_crds plugin to serve the records stored inside Kubernetes.

The operator only ensure (for now) the dns records's validity and state (active / inactive).
It also run the CoreDNS server, but it should be soon moved out and deployed by the operator.
The k8s_crds plugin ensure that valid dns apex are served if not defined via DNSRecord:
- it generates a valid NS record for each dns records zones (e.g ns0.dns.example.org)
- it generates a valid SOA record for each dns records zones


## TODOs:
- [ ] docs
- [ ] handle namespaces
- [ ] handle private IP address
- [ ] out of manager CoreDNS server
- [ ] CoreDNS server deployed by the manager
- [ ] find public CoreDNS server IP from LoadBalancer service
- [ ] add CoreDNS options (cache, log, etc.)
