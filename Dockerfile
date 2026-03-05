FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o proxy ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/proxy .
EXPOSE 1081
CMD ["./proxy"]
