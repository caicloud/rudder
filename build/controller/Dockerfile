#
# Copyright 2017 The Caicloud Authors.
#

FROM cargo.caicloudprivatetest.com/caicloud/alpine:3.7

RUN mkdir /data

COPY bin/controller /release

ENTRYPOINT ["/release"]
CMD ["-v4"]
