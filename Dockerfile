FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite libc6-compat
WORKDIR /root/
COPY --from=builder /app/server .
COPY frontend ./frontend

EXPOSE 8080
CMD ["./server"]
