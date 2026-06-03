FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY shared /app/shared
COPY services/notification-service /app/services/notification-service
WORKDIR /app/services/notification-service
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8004
CMD ["./server"]
