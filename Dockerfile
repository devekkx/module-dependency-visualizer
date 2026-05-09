FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w \
      -X 'github.com/devekkx/module-dependency-visualizer/internal/config.version=${VERSION}' \
      -X 'github.com/devekkx/module-dependency-visualizer/internal/config.commit=${COMMIT}' \
      -X 'github.com/devekkx/module-dependency-visualizer/internal/config.buildDate=${BUILD_DATE}'" \
    -o /mdv ./cmd/mdv

# Runtime image keeps Go so `go mod graph` works for Go projects.
FROM golang:1.24-alpine

COPY --from=builder /mdv /usr/local/bin/mdv

WORKDIR /work
ENTRYPOINT ["mdv"]
