FROM golang:latest AS builder

ENV GOOS=linux
ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app cmd/server/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app .