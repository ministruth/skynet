FROM debian:stable-slim as prod
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && apt-get clean
COPY bin /app
WORKDIR /app
CMD ["./skynet","run"]
