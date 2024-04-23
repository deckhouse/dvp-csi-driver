ARG BUILDER_CACHE_IMAGE=golang:1.21-alpine3.18
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
