# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -trimpath -o /sickgnal-server .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /sickgnal-server /usr/local/bin/sickgnal-server

EXPOSE 8080

ENTRYPOINT ["sickgnal-server"]
