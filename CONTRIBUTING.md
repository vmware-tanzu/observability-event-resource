# How to Contribute

We welcome third-party contributions to this tool!

If you're having a problem or believe you have found a bug, the best place to
start is to file an issue, providing as much detail as possible about your
environment. If you plan to share logs or other error details, please make sure
to review and remove any sensitive content before posting.

## Getting Started

If you want to go a step further and contribute changes, start by forking the
repository and cloning it to any directory on your local machine.

When you've finished with the change, open a pull request to discuss.

## Making Changes

Generally speaking, your change will likely be a code change, a documentation
change, or both.

### Changing the Source Code

This project uses Go 1.15+ and Go modules. The repo can be cloned to any
directory on your machine, and there is no need to set or modify your `$GOPATH`.

To run the tests, simply run `go test ./...` from the project root.

Any new changes should include tests where possible.

If your change is more than a trivial bug fix, it's usually helpful to open an
issue first to discuss the problem and your proposed solution. This ensures
we're aligned on the approach before you invest your time in writing the code.
