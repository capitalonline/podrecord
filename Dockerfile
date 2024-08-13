# Build the manager binary
FROM --platform=linux/amd64 golang:1.22-alpine AS builder

COPY ./ /workspace/podrecord
WORKDIR /workspace/podrecord

RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.io,direct && go mod tidy
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tool cmd/tools/main.go


FROM --platform=linux/amd64 ubuntu:20.04


WORKDIR /go/podrecord
COPY --from=builder /workspace/podrecord/manager .
COPY --from=builder /workspace/podrecord/tool .
#USER 65532:65532

CMD ["/go/podrecord/manager"]
