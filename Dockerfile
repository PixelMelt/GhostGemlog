FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o app ./main.go

FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
ENV GEMINI_HOST=0.0.0.0
ENV GEMINI_PORT=1965
COPY --from=builder /app/app /app/app
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD nc -z localhost $GEMINI_PORT || exit 1
CMD ["/app/app"]
