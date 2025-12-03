#!/bin/sh

# Start Go backend in background
cd /app
./main &

# Start Nginx in foreground
nginx -g 'daemon off;'
