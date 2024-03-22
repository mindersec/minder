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

ALTER TABLE profiles ADD COLUMN labels TEXT[] DEFAULT '{}';

-- This index is a bit besides the point of profile labels, but while profiling the searches
-- we noticed that the profile_id index was missing and it was causing a full table scan
-- which was more costly then the subsequent label filtering on the subset of profiles.
CREATE INDEX idx_entity_profiles_profile_id ON entity_profiles(profile_id);