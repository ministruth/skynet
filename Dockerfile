FROM ubuntu:latest
ARG TARGETARCH

RUN useradd -s /usr/sbin/nologin -r -c "Skynet User" skynet
COPY --chown=skynet:skynet release/$TARGETARCH /app
RUN chmod +x /app/skynet

WORKDIR /app
USER skynet
EXPOSE 8080
ENTRYPOINT ["./skynet"]
CMD ["run"]