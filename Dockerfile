FROM debian:stable-slim as prod
COPY bin /app
WORKDIR /app
CMD ["./skynet","run"]
