# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY mongoclient/ mongoclient/


# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use RedHat UniversalBaseImage minimal
FROM registry.access.redhat.com/ubi8-minimal:latest

ARG VERSION
ARG RELEASE

### Labels required by RedHat OpenShift
LABEL name="NVMesh Operator" \
      maintainer="support@excelero.com" \
      vendor="Excelero" \
      version="$VERSION" \
      release="$RELEASE" \
      summary="NVMesh Operator for deployment of NVMesh storage solution" \
      description="NVMesh Operator for Kubernetes and OpenShift"

COPY --from=builder /workspace/manager .
COPY resources/ resources/
COPY licenses/ licenses/

WORKDIR /

ENTRYPOINT ["/manager"]
