# -- Build stage --
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/webapp ./webapp

# -- Runtime stage --
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /bin/webapp .
COPY webapp/static ./static

EXPOSE 8083
CMD ["./webapp"]
