FROM golang:1.16

RUN mkdir -p /src/workspace
WORKDIR /src/workspace

COPY . .
RUN go install ./containers/runtime/xds/cmd/main.go

CMD ["bash"]
