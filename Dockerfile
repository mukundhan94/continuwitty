FROM golang:1.25-alpine AS builder

WORKDIR /srv

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY db ./db

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags="-s -w" -o /out/engram-api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /srv

COPY --from=builder /out/engram-api /usr/local/bin/engram-api
COPY --from=builder /srv/db ./db

RUN mkdir -p /srv/data

EXPOSE 8000

CMD ["engram-api"]
