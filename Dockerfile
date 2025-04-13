# Use the official Go image
FROM docker.io/golang:latest AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install Playwright CLI
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@latest

# Install Playwright browsers
RUN playwright install --with-deps

# Copy the source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 go build main.go

FROM docker.io/golang:latest

RUN apt update && apt upgrade -y && apt install -y curl tzdata
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@latest
RUN playwright install --with-deps



# Set the working directory
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/main .

# Copy the data directory
COPY --from=builder /app/data ./data

# RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

# Set the timezone
ENV TZ=America/Sao_Paulo

# Set the entry point for the container
ENTRYPOINT ["/app/main"]
