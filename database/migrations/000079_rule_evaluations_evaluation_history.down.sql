-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE public.rule_evaluations DROP COLUMN rule_entity_id;
ALTER TABLE public.rule_evaluations DROP COLUMN rule_instance_id;

ALTER TABLE latest_evaluation_statuses DROP CONSTRAINT latest_evaluation_statuses_profile_id_fkey;
ALTER TABLE latest_evaluation_statuses
    ADD CONSTRAINT latest_evaluation_statuses_profile_id_fkey
    FOREIGN KEY (profile_id)
    REFERENCES profiles(id);

-- recreate index with default name
DROP INDEX latest_evaluation_statuses_profile_id_idx;
CREATE INDEX idx_profile_id ON latest_evaluation_statuses(profile_id);

COMMIT;
