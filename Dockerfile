# Imagem base com Go
FROM golang:1.22-alpine

# Define diretório de trabalho
WORKDIR /app

# Copia arquivos do projeto
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compila a aplicação
RUN go build -o chat-server server.go

# Expondo a porta
EXPOSE 8080

# Comando para rodar
CMD ["./chat-server"]
