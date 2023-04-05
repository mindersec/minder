Continuous integration | License 
 ----------------------|---------
 [![Continuous integration](https://github.com/stacklok/mediator/actions/workflows/main.yml/badge.svg)](https://github.com/stacklok/mediator/actions/workflows/main.yml) | [![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)

# Mediator

Mediator is a platform to secure the software supply chain.
# API

The API is defined [here](https://github.com/stacklok/mediator/blob/main/proto/v1/mediator.proto).

It can be accessed over gRPC or HTTP using [gprc-gateway](https://grpc-ecosystem.github.io/grpc-gateway/).

## How to generate protobuf stubs

We use [buf](https://buf.build/docs/) to generate the gRPC / HTTP stubs (both protobuf and openAPI). 

To build the stubs, run:

```bash
buf generate
```

Should you introduce a new language, update the `buf.gen.yaml` file

New dependencies can be added to the `buf.yaml` file as follows:

```bash
version: v1
name: buf.build/stacklok/mediator
deps:
  - buf.build/googleapis/googleapis
  - buf.build/path/to/dependency
```

```bash
buf mod update
```
