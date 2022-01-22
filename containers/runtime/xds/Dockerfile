FROM golang:1.16

RUN mkdir /bootstrap
WORKDIR /src/workspace

COPY . .
RUN go install ./containers/runtime/xds/cmd/main.go
RUN mv /src/workspace/containers/runtime/xds/gRPC_bootstrap/bootstrap.json  /bootstrap/bootstrap.json

CMD ["bash"]
