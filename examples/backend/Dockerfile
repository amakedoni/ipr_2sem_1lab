FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backend .

FROM scratch
COPY --from=builder /app/backend /backend
CMD ["/backend"]
