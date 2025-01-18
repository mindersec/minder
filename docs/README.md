# Minder documentation

This directory contains the user documentation for Minder, hosted at
<https://mindersec.github.io>.

The docs are built with [Docusaurus](<[https://](https://docusaurus.io/)>), an
open source static website generator optimized for documentation use cases.

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

Run a preview server:

```bash
npm run start
```

Your browser should automatically open to <http://localhost:3000>

Build the docs:

```bash
cd docs
npm run build
```

Serve the docs

```bash
npm run serve -- --port 3001
```

Visit http://localhost:3001/ to view the docs.

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
