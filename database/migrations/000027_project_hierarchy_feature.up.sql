-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

--  hierarchy operations feature
INSERT INTO features(name, settings)
    VALUES ('project_hierarchy_operations_enabled', '{}')
    ON CONFLICT DO NOTHING;
