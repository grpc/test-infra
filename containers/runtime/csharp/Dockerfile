# Copyright 2020 gRPC authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM debian:stretch

RUN mkdir -p /src/workspace
WORKDIR /src/workspace

RUN apt-get update && apt-get install -y \
  autoconf \
  build-essential \
  clang \
  git \
  make \
  libtool \
  libgflags-dev \
  pkg-config && \
  apt-get clean

# The prerequisites installed below are heavily inspired by the C# testing image from grpc/grpc
# https://github.com/grpc/grpc/blob/master/tools/dockerfile/test/csharp_stretch_x64/Dockerfile

# cmake >=3.6 needed to build grpc_csharp_ext
RUN apt-get update && apt-get install -y curl && apt-get clean

# cmake >=3.6 needed to build grpc_csharp_ext
RUN apt-get update && apt-get install -y cmake && apt-get clean

# Install dotnet SDK
ENV DOTNET_SDK_VERSION 2.1.500
RUN curl -sSL -o dotnet.tar.gz https://dotnetcli.blob.core.windows.net/dotnet/Sdk/$DOTNET_SDK_VERSION/dotnet-sdk-$DOTNET_SDK_VERSION-linux-x64.tar.gz \
    && mkdir -p /usr/share/dotnet \
    && tar -zxf dotnet.tar.gz -C /usr/share/dotnet \
    && rm dotnet.tar.gz \
    && ln -s /usr/share/dotnet/dotnet /usr/bin/dotnet

# Trigger the population of the local package cache
ENV NUGET_XMLDOC_MODE skip
RUN mkdir warmup \
    && cd warmup \
    && dotnet new \
    && cd .. \
    && rm -rf warmup

# Python is needed to run the run_tests.py script (we use it to build C#)
RUN apt-get update && apt-get install -y \
  python-dev \
  python-setuptools \
  python-yaml && \
  apt-get clean

RUN curl https://bootstrap.pypa.io/get-pip.py | python2.7
RUN pip install futures==2.2.0 enum34==1.0.4 protobuf==3.5.2.post1 six==1.15.0 twisted==17.5.0

# Currently C# is little tricky to build with a single command, so we inject a script that can take care of the build.
RUN mkdir /build_scripts
ADD build_qps_worker.sh /build_scripts

CMD ["bash"]
