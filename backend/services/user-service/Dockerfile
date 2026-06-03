FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY shared /app/shared
COPY api-gateway /app/api-gateway
COPY services/user-service /app/services/user-service
WORKDIR /app/services/user-service
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8001
CMD ["./server"]
