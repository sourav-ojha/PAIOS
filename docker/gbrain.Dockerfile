FROM oven/bun:1

RUN bun install -g github:garrytan/gbrain

WORKDIR /brain
EXPOSE 7333

COPY gbrain-entrypoint.sh /usr/local/bin/gbrain-entrypoint.sh
RUN chmod +x /usr/local/bin/gbrain-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/gbrain-entrypoint.sh"]
