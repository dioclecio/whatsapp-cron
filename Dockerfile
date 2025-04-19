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
FROM mcr.microsoft.com/playwright:latest

# Instalar dependências adicionais necessárias


RUN apt update && apt upgrade -y 
RUN DEBIAN_FRONTEND=noninteractive TZ=America/Sao_Paulo apt install -y curl tzdata libnss3  libatk1.0-0  libatk-bridge2.0-0  libcups2  libdrm2  libxkbcommon0  libxcomposite1  libxdamage1  libxfixes3  libxrandr2  libgbm1  libpango-1.0-0  libasound2
RUN npm install -g npm
RUN npx playwright install chromium
RUN npx playwright install-deps chromium

# Set the working directory
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/main .

# Copy the data directory
COPY --from=builder /app/data ./data

# Set the timezone
ENV TZ=America/Sao_Paulo

# Set the entry point for the container
ENTRYPOINT ["/app/main"]
