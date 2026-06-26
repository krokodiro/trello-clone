FROM golang:1.26-alpine AS dev
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["go", "run", "./cmd/server"]

FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.21 AS runner
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /server .
COPY migrations ./migrations
EXPOSE 8080
CMD ["./server"]
