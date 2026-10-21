# Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API by default, can be overridden with --build-arg
ARG BINARY_PATH=cmd/api
RUN go build -o app ./${BINARY_PATH}

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]
