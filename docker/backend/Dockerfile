FROM golang:1.21-alpine

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build and run the application
RUN CGO_ENABLED=0 GOOS=linux go build -o tracker ./cmd/tracker

EXPOSE 8080

CMD ["./tracker"] 