# Build the manager binary
FROM golang:1.19 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
RUN mkdir -p cmd/k8s-dns/

# Copy the go source
COPY cmd/k8s-dns cmd/k8s-dns
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o k8s-dns -ldflags="-s -w" ./cmd/k8s-dns/

FROM alpine
WORKDIR /
COPY --from=builder /workspace/k8s-dns .

RUN apk add --no-cache ca-certificates

ENTRYPOINT ["/k8s-dns"]
