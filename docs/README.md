# Minder documentation

This directory contains the user documentation for Minder, hosted at
<https://mindersec.github.io>.

The docs are built with [Docusaurus](https://docusaurus.io/), an open source
static website generator optimized for documentation use cases.

## Contributing to docs

We welcome community contributions to the Minder documentation - if you find
something missing, wrong, or unclear, please let us know via an issue or open a
PR!

Please review the [style guide](./STYLE-GUIDE.md) for help with voice, tone, and
formatting.

## Building the docs locally

Start from the top level directory of the `minder` repository and generate the
CLI docs:

```bash
make cli-docs
```

Change to the docs directory:

```bash
cd docs
```

Run a preview server (this will automatically refresh most changes as you make
them):

```bash
npm run start
```

Your browser should automatically open to <http://localhost:3000>

Run a "production" build, this will also test for broken internal links:

```bash
npm run build
```

Serve the production build locally:

```bash
npm run serve -- --port 3001
```

Visit http://localhost:3001/ to view the build.

## Formatting

Before you submit a PR, please check for formatting and linting issues:

```bash
npm run prettier
npm run markdownlint
npm run eslint
```

To automatically fix issues:

```bash
npm run prettier:fix
npm run markdownlint:fix
npm run eslint:fix
```
