FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init provider_b && go mod tidy && go build -o server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8082
CMD ["./server"]
