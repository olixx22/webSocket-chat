#!/bin/sh
set -e

echo "Running database migrations..."
./migrator

echo "Starting messenger server..."
exec ./messenger-server
