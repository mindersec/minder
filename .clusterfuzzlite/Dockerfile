# SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

FROM gcr.io/oss-fuzz-base/base-builder-go
COPY . $SRC/minder
COPY .clusterfuzzlite/build.sh $SRC/build.sh
WORKDIR minder
