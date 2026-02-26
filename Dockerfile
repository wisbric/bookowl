# Build stage
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor \
    -ldflags="-s -w -X github.com/wisbric/bookowl/internal/version.Version=${VERSION} -X github.com/wisbric/bookowl/internal/version.Commit=${COMMIT}" \
    -o /bookowl ./cmd/bookowl

# Production stage
FROM gcr.io/distroless/static-debian12
COPY --from=build /bookowl /bookowl
COPY --from=build /app/migrations /migrations
ENTRYPOINT ["/bookowl"]
