# Use the official Go image as a base
FROM golang:1.20

# Set the working directory inside the container
WORKDIR /app

# Copy Go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project directory into the container
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the port your application uses
EXPOSE 8080

# Command to run your application
CMD ["./main"]
