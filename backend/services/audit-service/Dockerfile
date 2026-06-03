FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY shared /app/shared
COPY services/audit-service /app/services/audit-service
WORKDIR /app/services/audit-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8006
CMD ["./server"]
