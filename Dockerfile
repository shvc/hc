FROM alpine:3.17

COPY hc config.json /

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

WORKDIR /

ENV DEBUG=0 \
    ADDR=:80 \
    MSG='default container message' \
    CONFIG=/config.json

ENTRYPOINT ["/hc"]
