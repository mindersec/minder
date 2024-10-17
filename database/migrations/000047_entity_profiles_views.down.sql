-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER VIEW profiles_with_entity_profiles SET (security_invoker = false);

COMMIT;
