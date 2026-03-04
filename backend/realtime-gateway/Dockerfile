FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /realtime-gateway .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /realtime-gateway /realtime-gateway
EXPOSE 8080
ENTRYPOINT ["/realtime-gateway"]
