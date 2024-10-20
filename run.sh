#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define the stack name (e.g., dev, staging, production)
STACK_NAME="dev"

# Define the AWS region (adjust based on your needs)
AWS_REGION="ap-southeast-1"

# Check if Pulumi is installed
if ! command -v pulumi &> /dev/null
then
    echo "Pulumi could not be found, please install it first."
    exit 1
fi

# Ensure Go modules are set up and dependencies are installed
echo "Ensuring Go dependencies are installed..."
go mod tidy

# Set the AWS region in Pulumi config for the specified stack
echo "Setting AWS region in Pulumi stack configuration..."
pulumi config set aws:region $AWS_REGION --stack $STACK_NAME

# Select or create the stack
if pulumi stack ls | grep -q $STACK_NAME; then
    echo "Stack $STACK_NAME already exists. Selecting it..."
    pulumi stack select $STACK_NAME
else
    echo "Creating new Pulumi stack: $STACK_NAME..."
    pulumi stack init $STACK_NAME
fi

# Load environment variables from .env file
if [ -f .env ]; then
    echo "Loading environment variables from .env file..."
    export $(grep -v '^#' .env | xargs)
fi

# Check command line arguments
case "$1" in
    destroy)
        echo "Running pulumi destroy..."
        pulumi destroy --yes
        ;;
    cancel)
        echo "Cancelling any running pulumi operations..."
        # You can use pulumi cancel or manually find and kill the process
        pulumi cancel
        ;;
    *)
        echo "Running pulumi up..."
        go run main.go
        ;;
esac
