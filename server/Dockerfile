FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server .
RUN CGO_ENABLED=0 go build -o /consumer ./cmd/consumer

FROM alpine:3.19 AS api
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server .
EXPOSE 8080
CMD ["./server"]

FROM alpine:3.19 AS consumer
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /consumer .
CMD ["./consumer"]
