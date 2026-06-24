# syntax=docker/dockerfile:1

# --- build stage: compile a static binary with the pinned Go toolchain ---
FROM golang:1.23-alpine AS build
WORKDIR /src

# Resolve dependencies first so the layer is cached across source changes.
# go.sum is optional in the early foundations and tolerated with the glob.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/daedalus ./cmd/daedalus

# --- runtime stage: minimal image running as a non-root user ---
FROM alpine:3.20
RUN adduser -D -u 10001 daedalus
USER daedalus
COPY --from=build /out/daedalus /usr/local/bin/daedalus
ENTRYPOINT ["daedalus"]
