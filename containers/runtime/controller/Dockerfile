# Build the manager binary
FROM golang:1.14 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .
RUN chmod 777 /workspace/config/defaults.yaml

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/controller cmd/controller/main.go

FROM debian:buster
WORKDIR /workspace
COPY --from=builder /workspace .
CMD ["/workspace/bin/controller"]

