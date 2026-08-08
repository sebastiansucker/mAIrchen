#!/bin/sh
#
# Runs as the direct child of tini (PID 1, see docker/Dockerfile), which
# forwards TERM/INT here and reaps zombies. This script in turn supervises
# the Go backend and Nginx: it forwards shutdown signals to both, and if
# either process dies unexpectedly it tears down the other and exits
# non-zero so Docker notices the container is unhealthy and restarts it
# (see docker-compose.yml `restart: unless-stopped`) instead of leaving a
# half-dead container serving 502s indefinitely.

cd /app
./main &
GO_PID=$!

nginx -g 'daemon off;' &
NGINX_PID=$!

STOPPING=0
forward_signal() {
    STOPPING=1
    kill -TERM "$GO_PID" "$NGINX_PID" 2>/dev/null
}
trap forward_signal TERM INT

while kill -0 "$GO_PID" 2>/dev/null && kill -0 "$NGINX_PID" 2>/dev/null; do
    sleep 1
done

if [ "$STOPPING" -eq 0 ]; then
    echo "start-go.sh: a monitored process exited unexpectedly, shutting down container" >&2
    kill -TERM "$GO_PID" "$NGINX_PID" 2>/dev/null
fi

wait

if [ "$STOPPING" -eq 1 ]; then
    exit 0
fi
exit 1
