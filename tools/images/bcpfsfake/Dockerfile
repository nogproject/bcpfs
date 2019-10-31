# The Docker image `bcpfsfake` mimics the relevant aspects of the Linux system
# of the central file server.
#
# Tini is used to ensure quick exit on CTRL-C for any command.

# <https://hub.docker.com/_/debian/>
FROM debian:9.9@sha256:118cf8f3557e1ea766c02f36f05f6ac3e63628427ea8965fb861be904ec35a6f

LABEL maintainer "Steffen Prohaska <prohaska@zib.de>"

# <https://github.com/krallin/tini/releases>
ENV TINI_VERSION=v0.18.0

RUN set -x && \
    apt-get update && \
    apt-get -y install \
        acl \
        curl \
        gpg \
        quota \
    && \
    apt-get clean

# Tini
RUN set -x && \
    curl -sSL https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini -o /tmp/tini && \
    curl -sSL https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini.asc -o /tmp/tini.asc && \
    gpg --keyserver ha.pool.sks-keyservers.net --recv-keys 6380DC428747F6C393FEACA59A84159D7001A4E5 && \
    gpg --verify /tmp/tini.asc && \
    chmod a+x /tmp/tini && \
    mv /tmp/tini /usr/local/bin/tini

# Initialize Linux system to mimic the central file server.
COPY build /tmp/build
RUN set -x && \
    /tmp/build/addgroups && \
    rm -r /tmp/build

ENV PATH=/go/bin:${PATH}

# Tini with signal propagation to child group to ensure quick exit for any
# command.
ENTRYPOINT ["tini", "-g", "--"]
