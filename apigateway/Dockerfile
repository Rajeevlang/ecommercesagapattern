FROM golang:1.25.0-alpine AS builder
WORKDIR /app
COPY shared/ shared/
COPY apigateway/ apigateway/
RUN cd apigateway && CGO_ENABLED=0 GOOS=linux go build -o /server server.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
