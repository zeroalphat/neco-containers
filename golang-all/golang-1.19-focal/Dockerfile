# Go container

FROM --platform=$TARGETPLATFORM ghcr.io/cybozu/ubuntu-dev:20.04

ARG TARGETARCH
ARG GO_VERSION=1.19.9

ENV GOARCH=${TARGETARCH}
ENV GOPATH=/go
ENV PATH=/go/bin:/usr/local/go/bin:"$PATH"

CMD ["/bin/bash"]
