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

CREATE TABLE IF NOT EXISTS profile_selectors (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    entity entities, -- this is nullable since it can be applicable to all
    selector TEXT NOT NULL, -- CEL expression
    comment TEXT NOT NULL -- optional comment (can be empty string)
);

-- Ensure we have performant search based on profile_id
CREATE INDEX idx_profile_selectors_on_profile ON profile_selectors(profile_id);

COMMIT;
