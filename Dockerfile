# Build the manager binary
FROM golang:1.22 AS builder
#ARG TARGETOS
#ARG TARGETARCH


# Copy the Go Modules manifests
COPY ./ /workspace/podrecord
WORKDIR /workspace/podrecord
#COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
#RUN go mod download

# Copy the go source
#COPY cmd/main.go cmd/main.go
#COPY api/ api/
#COPY internal/controller/ internal/controller/
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.io,direct && go mod tidy
# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
#RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tool cmd/tools/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM alpine:3.14
FROM ubuntu:20.04

#FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/podrecord/manager .
COPY --from=builder /workspace/podrecord/tool .
USER 65532:65532

CMD ["/manager"]
