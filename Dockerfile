ARG BUILDER_CACHE_IMAGE=golang:1.21-alpine3.18@sha256:d390d5f2006c4c9cf19709d449a8463e7188643f444d06fcaa96f45b6c29363d
FROM $BUILDER_CACHE_IMAGE as builder

# Cache-friendly download modules.
ADD go.mod go.sum /app/
WORKDIR /app
RUN go mod download

# Build csi driver.
RUN rm -rf /app
ADD . /app
RUN GOOS=linux go build -o dvp-csi-driver ./cmd/dvp-csi-driver

FROM alpine:3.18@sha256:5292533eb4efd4b5cf35e93b5a2b7d0e07ea193224c49446c7802c19ee4f2da5
RUN apk add --no-cache e2fsprogs xfsprogs findmnt blkid
COPY --from=builder /app/dvp-csi-driver /

ENTRYPOINT ["/dvp-csi-driver"]
