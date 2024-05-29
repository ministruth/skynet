FROM ubuntu:latest

RUN useradd -s /usr/sbin/nologin -r -c "Skynet User" skynet
COPY --chown=skynet:skynet release/skynet /app

WORKDIR /app
USER skynet
EXPOSE 8080
ENTRYPOINT ["./skynet"]
CMD ["run"]