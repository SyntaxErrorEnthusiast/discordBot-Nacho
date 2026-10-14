#!/bin/sh
set -e

echo "Deploying slash commands..."
./deploy

echo "Starting bot..."
exec ./bot
