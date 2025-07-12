#!/bin/bash
set -e

# Default config file
CONFIG_FILE=${1:-"deploy-server-config.json"}

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file '$CONFIG_FILE' not found!"
    echo "Usage: $0 [config_file]"
    exit 1
fi

# Simple JSON parsing (no jq dependency)
parse_json() {
    local key=$1
    grep -o "\"$key\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "$CONFIG_FILE" | cut -d'"' -f4
}

# Parse configuration
KEY_PATH=$(parse_json "key_path")
REMOTE_USER=$(parse_json "user" | head -1)
REMOTE_HOST=$(parse_json "host")
REMOTE_DIR=$(parse_json "remote_dir")
BINARY_NAME=$(parse_json "binary_name")
SERVICE_NAME=$(parse_json "service_name")
SERVICE_DESCRIPTION=$(parse_json "service_description")
BUILD_PATH=$(parse_json "build_path")
CONFIG_FILE_NAME=$(parse_json "config_file")
GOOS=$(parse_json "goos")
GOARCH=$(parse_json "goarch")
SYSTEMD_USER=$(parse_json "user" | tail -1)
RESTART_POLICY=$(parse_json "restart_policy")
RESTART_SEC=$(parse_json "restart_sec")
AFTER_TARGET=$(parse_json "after")

# Expand tilde in KEY_PATH
KEY_PATH="${KEY_PATH/#\~/$HOME}"

# Set defaults
SERVICE_NAME=${SERVICE_NAME:-$BINARY_NAME}
SERVICE_DESCRIPTION=${SERVICE_DESCRIPTION:-"$BINARY_NAME Service"}
GOOS=${GOOS:-"linux"}
GOARCH=${GOARCH:-"amd64"}
SYSTEMD_USER=${SYSTEMD_USER:-"ubuntu"}
RESTART_POLICY=${RESTART_POLICY:-"on-failure"}
RESTART_SEC=${RESTART_SEC:-"5s"}
AFTER_TARGET=${AFTER_TARGET:-"network.target"}

# Validate required fields
if [ -z "$KEY_PATH" ] || [ -z "$REMOTE_USER" ] || [ -z "$REMOTE_HOST" ] || [ -z "$REMOTE_DIR" ] || [ -z "$BINARY_NAME" ]; then
    echo "Error: Missing required configuration fields"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE_NAME" ]; then
    echo "Error: $CONFIG_FILE_NAME file not found in current directory. This file is required for the application."
    exit 1
fi

# Build the application
echo "Building $BINARY_NAME for $GOOS/$GOARCH..."
GOOS=$GOOS GOARCH=$GOARCH go build -o $BINARY_NAME $BUILD_PATH

# Create directory on remote server
echo "Creating directory (if not exists)..."
ssh -i "$KEY_PATH" $REMOTE_USER@$REMOTE_HOST "sudo mkdir -p $REMOTE_DIR"

# Copy binary
echo "Copying binary to remote server..."
scp -i "$KEY_PATH" $BINARY_NAME $REMOTE_USER@$REMOTE_HOST:/home/$REMOTE_USER/

echo "Moving binary to target directory..."
ssh -i "$KEY_PATH" $REMOTE_USER@$REMOTE_HOST "sudo mv /home/$REMOTE_USER/$BINARY_NAME $REMOTE_DIR/"

# Copy config
echo "Copying config file to remote server..."
scp -i "$KEY_PATH" $CONFIG_FILE_NAME $REMOTE_USER@$REMOTE_HOST:/home/$REMOTE_USER/

echo "Moving config to target directory..."
ssh -i "$KEY_PATH" $REMOTE_USER@$REMOTE_HOST "sudo mv /home/$REMOTE_USER/$CONFIG_FILE_NAME $REMOTE_DIR/"

# Set up systemd service
echo "Setting up systemd service..."
ssh -i "$KEY_PATH" $REMOTE_USER@$REMOTE_HOST "sudo bash -c \"cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=$SERVICE_DESCRIPTION
After=$AFTER_TARGET

[Service]
Type=simple
User=$SYSTEMD_USER
WorkingDirectory=$REMOTE_DIR
ExecStart=sudo $REMOTE_DIR/$BINARY_NAME
Restart=$RESTART_POLICY
RestartSec=$RESTART_SEC

[Install]
WantedBy=multi-user.target
EOF\""

# Reload and restart service
echo "Reloading daemon and starting service..."
ssh -i "$KEY_PATH" $REMOTE_USER@$REMOTE_HOST "sudo systemctl daemon-reload && sudo systemctl enable $SERVICE_NAME.service && sudo systemctl restart $SERVICE_NAME.service"

# Check status
echo "Checking service status..."
ssh -i "$KEY_PATH" $REMOTE_USER@$REMOTE_HOST "sudo systemctl status $SERVICE_NAME.service"

echo "✅ Deployment complete!"