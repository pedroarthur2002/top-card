# === Etapa 1: build ===
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copia go.mod e go.sum primeiro para cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código do projeto
COPY . .

# Baixa dependências
RUN go mod tidy

# Compila o binário único a partir do main.go
RUN go build -o app ./cmd/main.go

# === Etapa 2: stage para testes ===
FROM golang:1.25-alpine AS tester
WORKDIR /app

# Instala ferramentas úteis para testes
RUN apk add --no-cache git

# Copia go.mod e go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código (necessário para testes)
COPY . .

# Baixa dependências
RUN go mod tidy

# Define o comando padrão para testes
CMD ["go", "test", "./test/...", "-v"]

# === Etapa 3: imagem final mínima ===
FROM alpine:3.18 AS runtime
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