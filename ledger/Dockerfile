# -- Build stage --
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=1 GOOS=linux go build -o /bin/ledger ./ledger

# -- Runtime stage --
FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite
WORKDIR /app
COPY --from=builder /bin/ledger .

EXPOSE 8080
CMD ["./ledger"]
