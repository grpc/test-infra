# How to Contribute

We definitely welcome your patches and contributions to gRPC! Please read the
gRPC organization's
[governance rules](https://github.com/grpc/grpc-community/blob/master/governance.md)
and
[contribution guidelines](https://github.com/grpc/grpc-community/blob/master/CONTRIBUTING.md)
before proceeding.

## Legal Requirements

In order to protect both you and the gRPC project, you will need to sign the
CNCF
[Contributor License Agreement](https://identity.linuxfoundation.org/projects/cncf)
before your PR can be merged.

## Communication

Trivial changes and small bug fixes do not need prior communication. You can
just submit a PR with minimum details. For larger PRs, we ask that before
contributing, please make the effort to coordinate with the maintainers of the
project via a Github issue or via
[grpcio](https://groups.google.com/forum/#!forum/grpc-io) mailing list. This
will prevent you from doing extra or redundant work that may or may not be
merged.

## Have Questions?

It is best to ask questions on forums other than Github repos. Github repos are
for filing issues and submitting PRs. You have a higher chance of getting help
from the community if you ask your questions on
[grpcio](https://groups.google.com/forum/#!forum/grpc-io) mailing list or
[Stackoverflow](https://stackoverflow.com/).

## Guidelines For Pull Requests

How to get your contributions merged smoothly and quickly.

- Create smaller PRs that are narrowly focused on addressing a single concern.
  We often times receive PRs that are trying to fix several things at a time,
  making the review process difficult. Create more PRs to address different
  concerns for faster resolution.

- Make sure to add new tests for bugs in order to catch regressions and to
  test any newly added functionality.

- For speculative changes, consider opening an issue and discussing it first.

- Provide a good PR description as a record of what change is being made and
  why it was made. Link to a GitHub issue if it exists.

- Don't fix code style and formatting unless you are already changing that
  line to address an issue. PRs with irrelevant changes won't be merged. If
  you do want to fix formatting or style, do that in a separate PR.

- Unless your PR is trivial, you should expect there will be reviewer comments
  that you'll need to address before merging. We expect you to be reasonably
  responsive to those comments, otherwise the PR will be closed after 2-3
  weeks of inactivity.

- Maintain clean commit history and use meaningful commit messages. PRs with
  messy commit history are difficult to review and won't be merged. Use
  `rebase -i upstream/master` to curate your commit history and/or to bring in
  latest changes from master but avoid rebasing in the middle of a code
  review.

- Keep your PR up to date with upstream/master. If there are merge conflicts,
  we can't really merge your change.

- All tests need to be passing before your change can be merged. We recommend
  you run tests locally before creating your PR to catch breakages early on.
  To test and check your commits locally, run the following:

  - `go test ./...` to run all tests
  - `go test -race ./...` to run tests in race mode
  - `go vet ./...` to find any functional issues
  - `golint ./...` to find any style issues (after installing [golint])

- Go code should comply with the [Effective Go] guide and try to avoid
  pitfalls in Go's [Code Review Comments]. It should be checked using [go vet]
  and formatted using [gofmt]. Comments should adhere to the
  [Documenting Go Code] post.

- Exceptions to the rules can be made if there's a compelling reason for doing
  so.

[golint]: https://github.com/golang/lint
[effective go]: https://golang.org/doc/effective_go.html
[code review comments]: https://github.com/golang/go/wiki/CodeReviewComments
[go vet]: https://golang.org/cmd/vet/
[gofmt]: https://blog.golang.org/gofmt
[documenting go code]: https://blog.golang.org/godoc
