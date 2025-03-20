#!/bin/bash
set -e

# Check for required files
if [[ ! -f .env ]]; then
    echo "Error: .env file is required but not found."
    exit 1
fi

if [[ ! -f web/.env ]]; then
    echo "Error: web/.env file is required but not found."
    exit 1
fi

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        echo "Error: Docker is not running"
        exit 1
    fi
}

# Function to deploy the application
deploy() {
    echo "Deploying tracker application..."
    
    if [ "$1" == "--rebuild" ]; then
        echo "Rebuilding containers..."
        docker-compose down
        docker-compose build --no-cache
    fi
    
    docker-compose up -d
    
    echo "Application deployed successfully!"
    echo "Frontend: http://localhost:8501"
    echo "Backend: http://localhost:8080"
}

# Main execution
check_docker
deploy "$1"
