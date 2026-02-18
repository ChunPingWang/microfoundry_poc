FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.version=${VERSION}" -o /mf ./cmd/mf

FROM alpine:3.21
RUN apk add --no-cache ca-certificates git
COPY --from=builder /mf /usr/local/bin/mf

EXPOSE 8080
ENTRYPOINT ["mf"]
