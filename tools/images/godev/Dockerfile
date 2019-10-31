# The `godev` Docker image contains a Go dev environment.
#
# The Dockerfile installs GRPC similar to the `grpc/go` image, see
# <https://hub.docker.com/r/grpc/go/~/dockerfile/>,
# <https://github.com/grpc/grpc-docker-library/blob/master/1.0/golang/Dockerfile>.
#
# Tini ensures quick exit on CTRL-C for any command; see
# <https://github.com/krallin/tini>.

# Base image <https://hub.docker.com/_/golang/>.
FROM golang:1.13.1@sha256:1b1246987ecb4939d9bd20f74eab25e8e2ad819aceea2d6b59869c82f85ef1de

LABEL maintainer "Steffen Prohaska <prohaska@zib.de>"

# ISO C standard locale, but with UTF-8.
ENV LC_ALL=C.UTF-8

# Enable Go modules for development.
#
# The Dockerfile uses `GO111MODULE=off go get` to disable modules when install
# tools, which is one of the alternatives discussed in the Go wiki,
# <https://github.com/golang/go/wiki/Modules#why-does-installing-a-tool-via-go-get-fail-with-error-cannot-find-main-module>.
ENV GO111MODULE=on

# <https://github.com/google/protobuf/releases>
# <https://github.com/golang/protobuf>
# <https://github.com/grpc/grpc-go/releases>
# <https://github.com/krallin/tini/releases>
#
# When upgrading `GRPC_GO_VERSION` or `PROTOC_GEN_GO_VERSION`, ensure that the
# `go.mod` versions are updated, too:
#
# ```
# make images
# ddev bash -c 'go get google.golang.org/grpc@${GRPC_GO_VERSION}'
# ddev bash -c 'go get github.com/golang/protobuf@${PROTOC_GEN_GO_VERSION}'
# ```
#
ENV \
    PROTOC_SHA256=213a591b42acb9e145ef9f3fe5f6037f14a14686065c508fcd547031243cc5d5 \
    PROTOC_VERSION=3.10.0 \
    PROTOC_GEN_GO_VERSION=v1.3.2 \
    GRPC_GO_VERSION=v1.24.0 \
    TINI_VERSION=v0.18.0

RUN set -x && \
    apt-get update && \
    apt-get -y install \
        acl \
        apt-utils \
        bzip2 \
        net-tools \
        netcat-openbsd \
        unzip \
    && \
    apt-get clean

# Tini
RUN set -x && \
    curl -sSL https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini -o /tmp/tini && \
    curl -sSL https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini.asc -o /tmp/tini.asc && \
    gpg --keyserver keyserver.ubuntu.com --recv-keys 6380DC428747F6C393FEACA59A84159D7001A4E5 && \
    gpg --verify /tmp/tini.asc && \
    chmod a+x /tmp/tini && \
    mv /tmp/tini /usr/local/bin/tini

# GRPC
RUN set -x && \
    mkdir /tmp/protoc && \
    cd /tmp/protoc && \
    curl -sSL https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip -o protoc.zip && \
    echo "${PROTOC_SHA256}  protoc.zip" | sha256sum -c && \
    unzip protoc.zip && \
    install -v -o root -g root -m 0755 bin/protoc /usr/local/bin/protoc && \
    find include -type f -exec \
        install -v -o root -g root -m 0644 -D '{}' '/usr/local/{}' ';' && \
    cd / && \
    rm -r /tmp/protoc

# GRCP Go, specific versions.
RUN set -x && \
    GO111MODULE=off go get -d \
        github.com/golang/protobuf/protoc-gen-go \
        google.golang.org/grpc \
    && \
    cd /go/src/github.com/golang/protobuf && \
    git checkout ${PROTOC_GEN_GO_VERSION} && \
    cd /go/src/google.golang.org/grpc && \
    git checkout ${GRPC_GO_VERSION} && \
    GO111MODULE=off go get \
        github.com/golang/protobuf/protoc-gen-go \
        google.golang.org/grpc \
    ;

# More Go tooling.
RUN set -x && \
    GO111MODULE=off go get \
        github.com/kisielk/errcheck \
    ;

# Certs tooling.
RUN set -x && \
    GO111MODULE=off go get \
        github.com/cloudflare/cfssl/cmd/cfssl \
        github.com/cloudflare/cfssl/cmd/cfssljson \
    ;

# Tini with signal propagation to child group to ensure quick exit for any
# command.
ENTRYPOINT ["tini", "-g", "--"]
