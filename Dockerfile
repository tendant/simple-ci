# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd/gateway

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and configs
COPY --from=builder /build/gateway .
COPY --from=builder /build/configs ./configs
COPY --from=builder /build/.env.example .

EXPOSE 8081

CMD ["./gateway"]
