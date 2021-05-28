# Building

In order to build a local binary for testing follow these steps:

1. [Install Go](https://golang.org/doc/install) >=1.16.4
1. [Install GoReleaser](https://goreleaser.com) install >= v0.166.0 (latest)
1. [Install golangci-lint](https://github.com/golangci/golangci-lint#install-golangci-lint) >=1.140.1

1. Clone the Circonus Unified Agent repository:

   ```sh
   cd ~/src
   git clone https://github.com/circonus-labs/circonus-unified-agent.git
   ```

1. Build a snapshot

   ```sh
   cd ~/src/circonus-unified-agent
   goreleaser build --rm-dist --snapshot
   ```

> Note: you can build for a specific target to avoid buidling all OS binaries by setting `GOOS` and using `--single-target`. For example: `GOOS=linux goreleaser --rm-dist --snapshot --single-target` will only produce the binaries for the `linux` target.
