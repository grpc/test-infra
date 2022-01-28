FROM golang:1.16

RUN mkdir /bootstrap
WORKDIR /src/workspace

COPY . .
RUN go install ./containers/runtime/xds/cmd/main.go

CMD ["bash"]
