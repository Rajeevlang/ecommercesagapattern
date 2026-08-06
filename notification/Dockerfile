FROM golang:1.25.0-alpine AS builder
WORKDIR /app
COPY shared/ shared/
COPY notification/ notification/
RUN cd notification && CGO_ENABLED=0 GOOS=linux go build -o /server cmd/server/main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /server /server
COPY notification/templates/ /templates/
EXPOSE 50056
ENTRYPOINT ["/server"]
