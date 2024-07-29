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

-- introduce some denormalization to simplify a common access pattern
-- namely: retrieving the latest rule statuses for a specific profile
-- A future PR will backfill this column and make it NOT NULL
ALTER TABLE latest_evaluation_statuses ADD COLUMN profile_id UUID REFERENCES profiles(id);
CREATE INDEX idx_profile_id ON latest_evaluation_statuses(profile_id);