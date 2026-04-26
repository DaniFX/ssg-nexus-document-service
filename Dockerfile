# Stage 1: Build
FROM golang:1.26.2-alpine AS builder

# Installa le dipendenze di sistema necessarie
RUN apk add --no-cache git

WORKDIR /app

# Copia i file dei moduli e scarica le dipendenze
COPY go.mod go.sum ./
RUN go mod download

# Copia il resto del codice
COPY . .

# Compila il binario (CGO_ENABLED=0 garantisce un binario statico per Alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -o /document-service ./cmd/document-service/main.go

# Stage 2: Final Image
FROM alpine:latest
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copia il binario dallo stage builder
COPY --from=builder /document-service .

# Espone la porta (Cloud Run usa solitamente la 8080)
EXPOSE 8080

# Variabile d'ambiente per Gin in produzione
ENV GIN_MODE=release

# Avvia il servizio
CMD ["./document-service"]