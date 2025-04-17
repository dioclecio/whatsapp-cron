# Use a imagem oficial do Go para a etapa de build
FROM docker.io/golang:latest AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o main main.go

# Use a imagem oficial do Playwright para a etapa final
FROM mcr.microsoft.com/playwright:v1.39.0-focal

# Instalar dependências adicionais necessárias
RUN apt update && apt upgrade -y && apt install -y curl tzdata

# Set the working directory
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/main .

# Copy the data directory
COPY --from=builder /app/data ./data

# Set the timezone
ENV TZ=America/Sao_Paulo

# Set the entry point for the container
ENTRYPOINT ["xvfb-run", "--server-args=-screen 0 1920x1080x24", "/app/main"]
