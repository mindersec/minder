-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Postgres can't remove a value for an enum type. So, we can't really
-- do a down migration. Instead, we'll just leave this here as a
-- reminder that we can't remove this value.
