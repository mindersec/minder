#
# Copyright 2023 Stacklok, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.21.3@sha256:b113af1e8b06f06a18ad41a6b331646dff587d7a4cf740f4852d16c49ed8ad73 AS builder
ENV APP_ROOT=/opt/app-root
ENV GOPATH=$APP_ROOT

WORKDIR $APP_ROOT/src/
ADD go.mod go.sum $APP_ROOT/src/
RUN go mod download

# Add source code
ADD ./ $APP_ROOT/src/

RUN CGO_ENABLED=0 go build -trimpath -o minder-server ./cmd/server

# Create a "nobody" non-root user for the next image by crafting an /etc/passwd
# file that the next image can copy in. This is necessary since the next image
# is based on scratch, which doesn't have adduser, cat, echo, or even sh.
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

RUN mkdir -p /app

FROM scratch

COPY --chown=65534:65534 --from=builder /app /app

WORKDIR /app

# Copy database directory and config. This is needed for the migration sub-command to work.
ADD --chown=65534:65534 ./cmd/server/kodata/config.yaml /app
ADD --chown=65534:65534 ./cmd/server/kodata/database/migrations /app/database/migrations

COPY --from=builder /opt/app-root/src/minder-server /usr/bin/minder-server

# Copy the certs from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the /etc_passwd file we created in the builder stage into /etc/passwd in
# the target stage. This creates a new non-root user as a security best
# practice.
COPY --from=builder /etc_passwd /etc/passwd

USER nobody

# Set the binary as the entrypoint of the container
ENTRYPOINT ["/usr/bin/minder-server"]
