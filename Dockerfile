ARG BUILDER_CACHE_IMAGE=golang:1.22-alpine3.18@sha256:d1a601b64de09e2fa38c95e55838961811d5ca11062a8f4230a5c434b3ae2a34
FROM $BUILDER_CACHE_IMAGE as builder

# Cache-friendly download modules.
ADD go.mod go.sum /app/
WORKDIR /app
RUN go mod download

# Build csi driver.
RUN rm -rf /app
ADD . /app
RUN GOOS=linux go build -o dvp-csi-driver ./cmd/dvp-csi-driver

FROM alpine:3.18
RUN apk add --no-cache e2fsprogs xfsprogs findmnt blkid
COPY --from=builder /app/dvp-csi-driver /

ENTRYPOINT ["/dvp-csi-driver"]
