# syntax=docker/dockerfile:1.7

# --platform=$BUILDPLATFORM: run the builder image natively on the CI runner.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

# Buildx sets these per target platform, for example linux/amd64 or linux/arm64.
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
# type=cache: keep downloaded Go modules in BuildKit cache between image builds.
# target=/go/pkg/mod: Go module download cache location.
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
# type=cache,target=/go/pkg/mod: reuse downloaded modules during compile.
# type=cache,target=/root/.cache/go-build: reuse Go compiler build cache.
# CGO_ENABLED=0: build a static binary that works in a small runtime image.
# GOOS/GOARCH: compile for the current Buildx target platform.
# -trimpath: remove local filesystem paths from the binary.
# -ldflags="-s -w": strip symbol/debug tables to reduce binary size.
# -o: write the server binary to the path copied into the runtime stage.
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/rivet-server ./cmd/rivet-server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=build /out/rivet-server /usr/local/bin/rivet-server

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/rivet-server"]
