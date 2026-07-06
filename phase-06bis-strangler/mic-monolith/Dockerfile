# syntax=docker/dockerfile:1.6
# MIC monolith - PHP 8.2-FPM + Nginx in a single image (didactic, simple).
FROM php:8.2-fpm-alpine

RUN --mount=type=secret,id=netskope_ca \
    if [ -f /run/secrets/netskope_ca ]; then \
      cat /run/secrets/netskope_ca >> /etc/ssl/certs/ca-certificates.crt; \
    fi \
 && apk add --no-cache nginx supervisor curl bash \
 && docker-php-ext-install pdo pdo_mysql

# nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# supervisord runs both nginx + php-fpm
RUN printf "%s\n" \
    "[supervisord]" \
    "nodaemon=true" \
    "logfile=/dev/null" \
    "logfile_maxbytes=0" \
    "" \
    "[program:php-fpm]" \
    "command=php-fpm -F" \
    "stdout_logfile=/dev/stdout" \
    "stdout_logfile_maxbytes=0" \
    "stderr_logfile=/dev/stderr" \
    "stderr_logfile_maxbytes=0" \
    "" \
    "[program:nginx]" \
    "command=nginx -g 'daemon off;'" \
    "stdout_logfile=/dev/stdout" \
    "stdout_logfile_maxbytes=0" \
    "stderr_logfile=/dev/stderr" \
    "stderr_logfile_maxbytes=0" \
    > /etc/supervisord.conf

# App
WORKDIR /app
COPY php-app/ /app/

EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --retries=10 \
  CMD curl -fsS http://127.0.0.1/api/dashboard/kpi >/dev/null || exit 1

CMD ["supervisord", "-c", "/etc/supervisord.conf"]
