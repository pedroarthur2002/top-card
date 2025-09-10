# === Etapa 1: build ===
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copia todo o código do projeto
COPY . .

# Baixa dependências
RUN go mod tidy

# Compila o binário único a partir do main.go
RUN go build -o app ./cmd/main.go

# === Etapa 2: imagem final mínima ===
FROM alpine:3.18

WORKDIR /app

# Copia o binário compilado do builder
COPY --from=builder /app/app .

# Garante permissão de execução
RUN chmod +x app

# Variáveis de ambiente padrão
ENV MODE=server
ENV SERVER_ADDR=127.0.0.1:8080

# Executa sempre o mesmo binário
ENTRYPOINT ["./app"]
