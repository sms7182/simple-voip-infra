FROM golang:1.26.8-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /sip-server ./cmd/main.go

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /sip-server /app/sip-server

EXPOSE 5060/udp

ENTRYPOINT ["/app/sip-server"]