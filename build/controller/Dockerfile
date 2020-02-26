FROM cargo.caicloud.xyz/library/debian:stretch

RUN mkdir /data

COPY bin/controller /release

ENTRYPOINT ["/release"]
CMD ["-v4"]
