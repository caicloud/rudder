#
# Copyright 2017 The Caicloud Authors.
#

FROM cargo.caicloud.io/caicloud/debian:jessie

# Set the default timezone to Shanghai
RUN echo "Asia/Shanghai" > /etc/timezone
RUN dpkg-reconfigure -f noninteractive tzdata

# Use aliyun source
RUN echo "deb http://mirrors.aliyun.com/debian/ jessie main non-free contrib" > /etc/apt/sources.list
RUN echo "deb http://mirrors.aliyun.com/debian/ jessie-proposed-updates main non-free contrib" >> /etc/apt/sources.list
RUN echo "deb-src http://mirrors.aliyun.com/debian/ jessie main non-free contrib" >>  /etc/apt/sources.list
RUN echo "deb-src http://mirrors.aliyun.com/debian/ jessie-proposed-updates main non-free contrib" >> /etc/apt/sources.list

RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  bash && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*


COPY bin/release /release
CMD ["/release", "-v4"]
