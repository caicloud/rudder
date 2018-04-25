#
# Copyright 2017 The Caicloud Authors.
#

FROM cargo.caicloudprivatetest.com/caicloud/debian:jessie

LABEL maintainer="Wei Guo <guowei@caicloud.io>"

RUN mkdir /data

COPY bin/linux_amd64/controller /release

ENTRYPOINT ["/release"]
CMD ["-v4"]


