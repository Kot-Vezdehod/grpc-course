FROM golang:1.27.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE
RUN CGO_ENABLED=0 go build -o /server ./cmd/${SERVICE}

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /server ./server

CMD ["./server"]
