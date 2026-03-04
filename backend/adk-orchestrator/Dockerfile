FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /adk-orchestrator .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /adk-orchestrator /adk-orchestrator
EXPOSE 8080
ENTRYPOINT ["/adk-orchestrator"]
