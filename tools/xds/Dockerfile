FROM golang:1.16

RUN mkdir -p /src/workspace
WORKDIR /src/workspace

COPY . .
RUN go install ./tools/xds/cmd/main.go -o .


CMD ["bash"]