#!/bin/bash
set -e

echo "Building tracker with RenderCV integration..."

if [[ ! -f .env ]]; then
    echo "Error: .env file is required but not found."
    exit 1
fi

docker build -t tracker-app -f Dockerfile .

if [[ "$1" == "--run" ]]; then
    echo "Running tracker container..."
    docker run -v $(pwd)/data/output:/app/data/output tracker-app
fi
