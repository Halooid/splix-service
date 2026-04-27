# Production stage
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY backend/go-shared /src/backend/go-shared
COPY backend/splix-service /src/backend/splix-service

WORKDIR /src/backend/splix-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/splix-service cmd/server/main.go

# Run stage
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/splix-service .
COPY --from=builder /src/backend/splix-service/internal/db/migration ./internal/db/migration

ENV GRPC_PORT=50051

EXPOSE 50051

CMD ["./splix-service"]
