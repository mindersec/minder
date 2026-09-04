-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Add `gitlab` provider type, so that a registered GitLab provider's
-- `implements` column can record that it supports the `gitlab` trait,
-- matching what `provifv1.Provider.CanImplement` already gates at
-- evaluation time.
ALTER TYPE provider_type ADD VALUE 'gitlab';
