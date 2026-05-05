FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/rivet-server ./cmd/rivet-server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=build /out/rivet-server /usr/local/bin/rivet-server

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/rivet-server"]
