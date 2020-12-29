# Kubernetes DNS Operator

The DNS Operator define a new resource: DNSRecord. This resources is then used by a CoreDNS plugin to 
service the records stored inside Kubernetes.


## TODOs:
- [ ] docs
- [ ] handle namespaces
- [ ] handle private IP address
- [ ] out of manager CoreDNS server
- [ ] CoreDNS server deployed by the manager
- [ ] find public CoreDNS server IP from LoadBalancer service
