FROM golang:1.23@sha256:c2d828fd49c47ed2b9192d2dbffed83052d8a21af465056d732c3de0d756f217 AS base

ENV GO111MODULE=on

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

FROM base as builder
COPY pkg pkg
COPY main.go main.go
COPY Makefile Makefile

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on make build

FROM gcr.io/distroless/static:nonroot
WORKDIR /
USER nonroot:nonroot
COPY --chown=nonroot:nonroot --from=builder /workspace/bin/s3-webserver .
COPY LICENSE LICENSE

ENTRYPOINT [ "/s3-webserver" ]
