FROM golang:1.25.0-alpine AS builder
WORKDIR /app
COPY shared/ shared/
COPY payment/ payment/
RUN cd payment && CGO_ENABLED=0 GOOS=linux go build -o /server cmd/server/main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /server /server
EXPOSE 50055
ENTRYPOINT ["/server"]
