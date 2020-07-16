# Clone

Clone is a container that clones a GitHub repository into its working directory
and checks out a specific commit, branch or tag. It is intended to be used as an
init container with a volume mount.

The environment variables `$CLONE_REPO` and `$CLONE_GIT_REF` set the address of
the repository and the commit, branch or tag to checkout. `$CLONE_REPO` should
be a URL with a `.git` extension, like:
`https://github.com/grpc/test-infra.git`.

This version of clone does not support SSH and is tested with HTTP/HTTPS.
