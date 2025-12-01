# Multi-Stage Dockerfile - Frontend + Backend in einem Container
FROM node:18-alpine as frontend-build
WORKDIR /frontend
COPY frontend/ .

# Backend Stage
FROM python:3.11-slim

WORKDIR /app

# System-Dependencies für Nginx
RUN apt-get update && \
    apt-get install -y nginx && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Python Dependencies installieren
COPY ../backend/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Backend-Code und Grundwortschatz kopieren
COPY ../backend/ .

# Frontend-Dateien kopieren
COPY ../frontend/ /var/www/html/

# Nginx-Konfiguration
COPY docker/nginx-combined.conf /etc/nginx/sites-available/default

# Startup-Script erstellen
# Backend läuft nur auf 127.0.0.1 (localhost) - nicht von außen erreichbar!
RUN echo '#!/bin/bash\n\
nginx\n\
uvicorn main:app --host 127.0.0.1 --port 8000\n\
' > /start.sh && chmod +x /start.sh

# Nur Port 80 nach außen (Nginx), Backend-Port 8000 ist intern
EXPOSE 80

CMD ["/start.sh"]
