-- Copyright 2024 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

BEGIN;

ALTER TABLE public.rule_evaluations DROP COLUMN rule_entity_id;

ALTER TABLE latest_evaluation_statuses DROP CONSTRAINT latest_evaluation_statuses_profile_id_fkey;
ALTER TABLE latest_evaluation_statuses
    ADD CONSTRAINT latest_evaluation_statuses_profile_id_fkey
    FOREIGN KEY (profile_id)
    REFERENCES profiles(id);

-- recreate index with default name
DROP INDEX latest_evaluation_statuses_profile_id_idx;
CREATE INDEX idx_profile_id ON latest_evaluation_statuses(profile_id);

COMMIT;