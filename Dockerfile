# Use the official Go image
FROM golang:latest AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy

# Copy the source code
COPY . .

# Build the Go application
RUN go build main.go

# Use a lightweight image to run the application
FROM alpine:latest

# Install Chromium and ChromeDriver
RUN apk update && apk add chromium chromium-chromedriver curl

# Set the working directory
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/main .

# Copy the data directory
COPY --from=builder /app/data ./data

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2


# Set the entry point for the container
ENTRYPOINT ["/app/main"]
