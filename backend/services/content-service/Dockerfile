FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY shared /app/shared
COPY services/content-service /app/services/content-service
WORKDIR /app/services/content-service
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8002
CMD ["./server"]
