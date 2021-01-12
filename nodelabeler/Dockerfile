# Default base images for builder and release images, can be overridden during build
ARG BUILDER_IMAGE=gcr.io/gcp-runtimes/go1-builder:1.13
ARG RELEASE_IMAGE=scratch

# Create the binary in a builder image, so the release image can be kept small
FROM ${BUILDER_IMAGE} as builder

ENV PATH "/usr/local/go/bin:$PATH"

SHELL [ "/bin/bash", "-cex" ]
WORKDIR /usr/src/nodelabeler

# Take advantage of caching for dependency acquisition
COPY go.mod go.sum /usr/src/nodelabeler/
RUN go mod download

# Create a static binary - because net package is used, a dynamic binary will not work with scratch
COPY nodelabeler/main.go /usr/src/nodelabeler/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-w -s -extldflags '-static'" -o /go/bin/nodelabeler

# Transfer the binary to a clean release image
FROM ${RELEASE_IMAGE}

LABEL org.opencontainers.image.authors='airship-discuss@lists.airshipit.org, irc://#airshipit@freenode' \
      org.opencontainers.image.url='https://airshipit.org' \
      org.opencontainers.image.documentation='https://docs.airshipit.org/nodelabeler/' \
      org.opencontainers.image.source='https://opendev.org/airship/nodelabeler' \
      org.opencontainers.image.vendor='The Airship Authors' \
      org.opencontainers.image.licenses='Apache-2.0'

COPY --from=builder /go/bin/nodelabeler /go/bin/nodelabeler
USER 65534
CMD ["./go/bin/nodelabeler"]