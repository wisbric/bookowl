# Build stage
FROM golang:1.25-alpine AS build
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./

# Drop local replace directive; fetch private core module via injected token
RUN --mount=type=secret,id=github_token \
    git config --global url."https://x-access-token:$(cat /run/secrets/github_token)@github.com/".insteadOf "https://github.com/" && \
    GOPRIVATE=github.com/wisbric/* go mod edit -dropreplace=github.com/wisbric/core && \
    go mod download

COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/wisbric/bookowl/internal/version.Version=${VERSION} -X github.com/wisbric/bookowl/internal/version.Commit=${COMMIT}" \
    -o /bookowl ./cmd/bookowl

# Production stage
FROM gcr.io/distroless/static-debian12
COPY --from=build /bookowl /bookowl
COPY --from=build /app/migrations /migrations
ENTRYPOINT ["/bookowl"]
