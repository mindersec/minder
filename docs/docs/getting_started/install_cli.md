---
title: Install Minder CLI
sidebar_position: 10
---

# Installing the Minder CLI

The open source Minder CLI can communicate with either [Minder Cloud](https://cloud.stacklok.com/), Stacklok's free-to-use, cloud-hosted version of Minder, or with an on-premises version of Minder that has been [set up independently](../run_minder_server/run_the_server).

The `minder` CLI is built for `amd64` and `arm64` architectures on Windows, MacOS, and Linux.

You can install `minder` using one of the following methods:

## macOS

The easiest way to install `minder` for macOS systems is through [Homebrew](https://brew.sh/):

```bash
brew install stacklok/tap/minder
```

Alternatively, you can [download a `.tar.gz` release](https://github.com/stacklok/minder/releases) and unpack it with the following:

```bash
tar -xzf minder_${RELEASE}_darwin_${ARCH}.tar.gz minder
xattr -d com.apple.quarantine minder
```

## Windows

For Windows, the built-in `winget` tool is the easiest way to install `minder`:

```bash
winget install stacklok.minder
```

Alternatively, you can [download a zipfile containing the `minder` CLI](https://github.com/stacklok/minder/releases) and install the binary yourself.

## Linux

We provide pre-built static binaries for Linux at: https://github.com/stacklok/minder/releases.

## Building from source

You can also build the `minder` CLI from source using `go install github.com/stacklok/minder/cmd/cli@latest`, or by [following the build instructions in the repository](https://github.com/stacklok/minder#build-from-source).
