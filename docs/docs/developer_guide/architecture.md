---
title: Architecture overview
sidebar_position: 60
---

# System Architecture

While it is built as a single container, Minder implements several external
interfaces for different components. In addition to the GRPC and HTTP service
ports, it also leverages the [watermill library](https://watermill.io) to queue
and route events within the application.

The following is a high-level diagram of the Minder architecture

```mermaid
flowchart LR
    subgraph minder
        %% flow from top to bottom
        direction TB

        grpc>GRPC endpoint]
        click grpc "/api" "GRPC auto-generated documentation"
        web>HTTP endpoint]
        click web "https://github.com/stacklok/minder/blob/main/internal/controlplane/server.go#L210" "Webserver URL registration code"
        events("watermill")
        click events "https://watermill.io/docs" "Watermill event processing library"

        handler>Event handlers]
        click handler "https://github.com/stacklok/minder/blob/main/cmd/server/app/serve.go#L69" "Registered event handlers"
    end

    cloud([GitHub])
    cli("<code>minder</code> CLI")
    click cli "https://github.com/stacklok/minder/tree/main/cmd/cli"

    db[(Postgres)]
    click postgres "/db/minder_db_schema" "Database schema"

    cli --> grpc
    cli --OAuth--> web
    cloud --> web

    grpc --> db
    web --> db

    web --> events

    events --> handler
```
