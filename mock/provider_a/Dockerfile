FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init provider_a && go mod tidy && go build -o server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8081
CMD ["./server"]
