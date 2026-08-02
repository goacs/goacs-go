# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/goacs .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/goacs ./goacs
COPY contrib/database ./contrib/database
# Default configuration so the image runs out of the box; override by mounting
# your own .env at /app/.env for a real deployment (see README for the full list
# of variables, e.g. MYSQL_*, JWT_SECRET, CORS_ALLOWED_ORIGINS).
COPY .env.example ./.env

RUN mkdir -p ./storage
VOLUME ["/app/storage"]

EXPOSE 8085

ENTRYPOINT ["./goacs"]
