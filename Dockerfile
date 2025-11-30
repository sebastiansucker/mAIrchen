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
COPY backend/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Backend-Code und Grundwortschatz kopieren
COPY backend/ .
COPY gws.md /app/gws.md

# Frontend-Dateien kopieren
COPY frontend/ /var/www/html/

# Nginx-Konfiguration
COPY nginx-combined.conf /etc/nginx/sites-available/default

# Startup-Script erstellen
RUN echo '#!/bin/bash\n\
nginx\n\
uvicorn main:app --host 0.0.0.0 --port 8000\n\
' > /start.sh && chmod +x /start.sh

EXPOSE 80 8000

CMD ["/start.sh"]
