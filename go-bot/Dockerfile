# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot && \
    CGO_ENABLED=0 GOOS=linux go build -o deploy ./cmd/deploy

# ---- Final image ----
FROM alpine:3.20

# ca-certificates: HTTPS calls to Google APIs and Discord
# poppler-utils: provides pdftoppm for PDF-to-image rendering
RUN apk --no-cache add ca-certificates poppler-utils

WORKDIR /app

COPY --from=builder /app/bot .
COPY --from=builder /app/deploy .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

CMD ["./entrypoint.sh"]
