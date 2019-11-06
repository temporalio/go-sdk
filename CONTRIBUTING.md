# Developing cadence-client

This doc is intended for contributors to `cadence-client` (hopefully that's you!)

**Note:** All contributors also need to fill out the [Uber Contributor License Agreement](http://t.uber.com/cla) before we can merge in any of your changes

## Development Environment

* Go. Install on OS X with `brew install go`. The minimum required Go version is 1.10.
* `thrift`. Install on OS X with `brew install thrift`.
* `thrift-gen`. Install with `go get github.com/uber/tchannel-go/thrift/thrift-gen`.

## Checking out the code

Make sure the repository is cloned to the correct location:
(Note: the path is `go.temporal.io/temporal/` instead of github repo)

```bash
go get go.temporal.io/temporal/...
cd $GOPATH/src/go.temporal.io/temporal
```

## Dependency management

Dependencies are tracked via `Gopkg.toml`. If you're not familiar with `dep`,
read the [docs](https://https://golang.github.io/dep/docs/introduction.html).

## Licence headers

This project is Open Source Software, and requires a header at the beginning of
all source files. To verify that all files contain the header execute:

```lang=bash
make copyright
```

## Commit Messages

Overcommit adds some requirements to your commit messages. At Uber, we follow the
[Chris Beams](http://chris.beams.io/posts/git-commit/) guide to writing git
commit messages. Read it, follow it, learn it, love it.

## Testing

Run all the tests with coverage and race detector enabled:

```bash
make test
```
