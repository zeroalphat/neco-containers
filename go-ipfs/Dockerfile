# referring to go-ipfs Dockerfile(https://github.com/ipfs/go-ipfs/blob/master/Dockerfile)
FROM quay.io/cybozu/golang:1.17-focal AS build-idserver
WORKDIR /idserver
COPY idserver/go.mod /idserver/
COPY idserver/go.sum /idserver/
RUN go mod download 

COPY idserver/*.go /idserver/
RUN go build -o /idserver
