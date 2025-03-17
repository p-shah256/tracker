FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tracker ./cmd/tracker


FROM python:3.12-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir rendercv
WORKDIR /app

COPY --from=builder /app/tracker /app/tracker
COPY ./configs ./configs
COPY ./data ./data
COPY .env .env

RUN mkdir -p /app/data/output/rendercv
RUN chmod +x /app/tracker
CMD ["/app/tracker"]
