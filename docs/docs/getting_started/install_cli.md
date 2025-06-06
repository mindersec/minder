---
title: Installing the Minder CLI
sidebar_label: Install Minder CLI
sidebar_position: 10
---

The open source Minder CLI can communicate with either
[the free public instance provided by Custcodian](../../#minder-public-instance),
or with a [self managed server](../run_minder_server/run_the_server).

The `minder` CLI is built for `amd64` and `arm64` architectures on Windows,
macOS, and Linux.

You can install `minder` using one of the following methods:

## macOS

The easiest way to install `minder` for macOS systems is through
[Homebrew](https://brew.sh/):

```bash
brew install minder
```

Alternatively, you can
[download a `.tar.gz` release](https://github.com/mindersec/minder/releases) and
unpack it with the following:

```bash
tar -xzf minder_${RELEASE}_darwin_${ARCH}.tar.gz minder
xattr -d com.apple.quarantine minder
```

## Windows

For Windows, the built-in `winget` tool is the easiest way to install `minder`:

```bash
winget install mindersec.minder
```

Alternatively, you can
[download a zip file containing the `minder` CLI](https://github.com/mindersec/minder/releases)
and install the binary yourself.

## Linux

We provide pre-built static binaries for Linux at
[https://github.com/mindersec/minder/releases](https://github.com/mindersec/minder/releases).

## Building from source

You can also build the `minder` CLI from source using
`go install github.com/mindersec/minder/cmd/cli@latest`, or by
[following the build instructions in the repository](https://github.com/mindersec/minder#build-from-source).
