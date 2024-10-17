-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TYPE provider_type ADD VALUE 'image-lister';

-- Add `ghcr` and `dockerhub` provider classes
ALTER TYPE provider_class ADD VALUE 'ghcr';
ALTER TYPE provider_class ADD VALUE 'dockerhub';
