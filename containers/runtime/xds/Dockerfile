FROM golang:1.16

RUN mkdir -p /src/workspace
WORKDIR /src/workspace

# when BREAK_CACHE arg is set to a random value (e.g. by "--build-arg BREAK_CACHE=$(uuidgen)"),
# it makes sure the docker cache breaks at this command, and all the following
# commands in this Dockerfile will be forced to re-run on each build.
# This is important to ensure we always clone the repository even if "GITREF" stays unchanged
# (important e.g. when GITREF=master, when the clone command could get cached and
# we'd end up with a stale repository).
ARG BREAK_CACHE

COPY . .
RUN go install ./containers/runtime/xds/cmd/main.go

CMD ["bash"]
