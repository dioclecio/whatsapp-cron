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
RUN go build main.go

# Use a imagem oficial do Playwright para a etapa final
FROM docker.io/golang:latest AS final
# FROM mcr.microsoft.com/playwright:v1.52.0-noble

# Set the timezone
ENV TZ=America/Sao_Paulo

# RUN apt update && apt upgrade -y && apt install -y curl tzdata
# RUN apt update && apt upgrade -y 
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@latest
RUN playwright install chromium-headless-shell --with-deps

# Set the working directory
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/main .

# Copy the data directory
COPY --from=builder /app/data ./data



# Set the entry point for the container
ENTRYPOINT ["/app/main"]
