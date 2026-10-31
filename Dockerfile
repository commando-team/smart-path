FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app ./cmd \
    && go clean -cache -modcache

FROM alpine:latest

WORKDIR /smart-path

COPY --from=builder /build/app ./app
COPY --from=builder /build/db/migrations ./db/migrations

EXPOSE 8080

CMD ["./app"]