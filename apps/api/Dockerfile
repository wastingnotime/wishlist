# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wishlist-api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 wishlist \
    && adduser -S -D -H -u 10001 -G wishlist wishlist
COPY --from=build /out/wishlist-api /usr/local/bin/wishlist-api

ARG BUILD_DATE=unknown
ARG IMAGE_SOURCE=https://github.com/wastingnotime/wishlist
ARG VCS_REF=unknown
LABEL org.opencontainers.image.title="wishlist-api" \
      org.opencontainers.image.description="WNT Wishlist API" \
      org.opencontainers.image.source="${IMAGE_SOURCE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}"

USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -q -O - http://127.0.0.1:8080/readyz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/usr/local/bin/wishlist-api"]
