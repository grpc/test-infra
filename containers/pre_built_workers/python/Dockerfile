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

# See https://github.com/grpc/grpc/blob/master/tools/dockerfile/README.md
FROM us-docker.pkg.dev/grpc-testing/testing-images-public/bazel:ac0cf07a320bb1f337986fd312a1785614cc1e4e@sha256:4b64295380d6c951f3761273344e256f44daab35dd7182ff431a1802695368da

RUN mkdir -p /pre
WORKDIR /pre

ARG REPOSITORY=grpc/grpc
ARG GITREF=master
# when BREAK_CACHE arg is set to a random value (e.g. by "--build-arg BREAK_CACHE=$(uuidgen)"),
# it makes sure the docker cache breaks at this command, and all the following
# commands in this Dockerfile will be forced to re-run on each build.
# This is important to ensure we always clone the repository even if "GITREF" stays unchanged
# (important e.g. when GITREF=master, when the clone command could get cached and
# we'd end up with a stale repository).
ARG BREAK_CACHE

RUN git clone https://github.com/$REPOSITORY.git .
# checkout, but skip updating the submodules since they're not use by bazel build
RUN git checkout $GITREF

# Save commit sha for debug use
RUN echo 'COMMIT SHA' > GRPC_GIT_COMMIT.txt
RUN git rev-parse $GITREF >> GRPC_GIT_COMMIT.txt

RUN mkdir /executables
RUN bazel --output_user_root=/executables build -c opt //src/python/grpcio_tests/tests/qps:qps_worker
RUN bazel --output_user_root=/executables build -c opt //src/python/grpcio_tests/tests_aio/benchmark:worker

RUN mkdir -p /execute && \
    cp -r -L /pre/bazel-bin/src/python/grpcio_tests/tests/qps/qps_worker /execute/qps_worker && \
    cp -r -L /pre/bazel-bin/src/python/grpcio_tests/tests/qps/qps_worker.runfiles /execute/qps_worker.runfiles && \
    cp -r -L /pre/bazel-bin/src/python/grpcio_tests/tests_aio/benchmark/worker /execute/benchmark_worker && \
    cp -r -L /pre/bazel-bin/src/python/grpcio_tests/tests_aio/benchmark/worker.runfiles /execute/benchmark_worker.runfiles && \
    cp /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

FROM python:3.9-slim-bullseye

RUN ln -s /usr/local/bin/python3 /usr/bin/python3 || true && \
    ln -s /usr/local/bin/python /usr/bin/python || true

COPY --from=0 /execute /execute

# Recreate the symlinks inside the virtualenvs to fix Python's virtualenv detection
RUN rm -f /execute/qps_worker.runfiles/_main/src/python/grpcio_tests/tests/qps/_qps_worker.venv/bin/python3 && \
    ln -s /execute/qps_worker.runfiles/rules_python++python+python_3_11_x86_64-unknown-linux-gnu/bin/python3 \
          /execute/qps_worker.runfiles/_main/src/python/grpcio_tests/tests/qps/_qps_worker.venv/bin/python3

RUN rm -f /execute/benchmark_worker.runfiles/_main/src/python/grpcio_tests/tests_aio/benchmark/_worker.venv/bin/python3 && \
    ln -s /execute/benchmark_worker.runfiles/rules_python++python+python_3_11_x86_64-unknown-linux-gnu/bin/python3 \
          /execute/benchmark_worker.runfiles/_main/src/python/grpcio_tests/tests_aio/benchmark/_worker.venv/bin/python3

# Prioritize the main repository root (_main) in PYTHONPATH so top-level 'src' imports are never shadowed
ENV PYTHONPATH="/execute/qps_worker.runfiles/_main:/execute/benchmark_worker.runfiles/_main"

WORKDIR /execute

CMD ["bash"]
