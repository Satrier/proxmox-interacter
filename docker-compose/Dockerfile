FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /src

RUN git clone https://github.com/Satrier/proxmox-interacter .

RUN go mod download
RUN go build -o /app/proxmox-interacter ./cmd/...


FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/proxmox-interacter .
COPY --from=builder /src/templates ./templates

ENTRYPOINT ["./proxmox-interacter", "--config", "/app/config.toml"]