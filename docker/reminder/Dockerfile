# SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

FROM index.docker.io/library/golang:1.23.4@sha256:574185e5c6b9d09873f455a7c205ea0514bfd99738c5dc7750196403a44ed4b7 AS builder
ENV APP_ROOT=/opt/app-root
ENV GOPATH=$APP_ROOT

WORKDIR $APP_ROOT/src/
ADD go.mod go.sum $APP_ROOT/src/
RUN go mod download

# Add source code
ADD ./ $APP_ROOT/src/

RUN CGO_ENABLED=0 go build -trimpath -o reminder ./cmd/reminder

# Create a "nobody" non-root user for the next image by crafting an /etc/passwd
# file that the next image can copy in. This is necessary since the next image
# is based on scratch, which doesn't have adduser, cat, echo, or even sh.
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

RUN mkdir -p /app

FROM scratch

COPY --chown=65534:65534 --from=builder /app /app

WORKDIR /app

COPY --from=builder /opt/app-root/src/reminder /usr/bin/reminder

# Copy the certs from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the /etc_passwd file we created in the builder stage into /etc/passwd in
# the target stage. This creates a new non-root user as a security best
# practice.
COPY --from=builder /etc_passwd /etc/passwd

USER nobody

# Set the binary as the entrypoint of the container
ENTRYPOINT ["/usr/bin/reminder"]
