# Build stage
ARG ALPINE_VERSION=3.19

FROM docker.io/library/alpine:${ALPINE_VERSION} AS builder
ARG TARGETARCH
ARG TARGETOS
ARG TARGETVARIANT
COPY dist/shiori_${TARGETOS}_${TARGETARCH}${TARGETVARIANT}/shiori /usr/bin/shiori
RUN apk add --no-cache ca-certificates tzdata && \
    chmod +x /usr/bin/shiori && \
    rm -rf /tmp/*

# Server image
FROM scratch

ENV PORT=8080
ENV SHIORI_DIR=/shiori
WORKDIR ${SHIORI_DIR}

LABEL org.opencontainers.image.source="https://github.com/go-shiori/shiori"
LABEL maintainer="Felipe Martin <github@fmartingr.com>"

COPY --from=builder /tmp /tmp
COPY --from=builder /usr/bin/shiori /usr/bin/shiori
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE ${PORT}

<<<<<<< HEAD
||||||| parent of ebc1ce2 (root)
FROM ghcr.io/ghcri/alpine:3.16
LABEL org.opencontainers.image.source https://github.com/go-shiori/shiori
COPY --from=builder /src/shiori /usr/bin/
RUN addgroup -g 1000 shiori \
    && adduser -D -h /shiori -g '' -G shiori -u 1000 shiori
USER shiori
WORKDIR /shiori
EXPOSE 8080
ENV SHIORI_DIR /shiori/
=======
FROM ghcr.io/ghcri/alpine:3.16
LABEL org.opencontainers.image.source https://github.com/go-shiori/shiori
COPY --from=builder /src/shiori /usr/bin/
WORKDIR /shiori
EXPOSE 8080
ENV SHIORI_DIR /shiori/
>>>>>>> ebc1ce2 (root)
ENTRYPOINT ["/usr/bin/shiori"]
CMD ["server"]
