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


ALTER TABLE evaluation_statuses ALTER COLUMN most_recent_evaluation TYPE TIMESTAMP;
ALTER TABLE evaluation_statuses ALTER COLUMN most_recent_evaluation SET DEFAULT NOW();
ALTER TABLE evaluation_statuses ALTER COLUMN evaluation_times TYPE TIMESTAMP[];
ALTER TABLE evaluation_statuses ALTER COLUMN evaluation_times SET DEFAULT ARRAY[NOW()]::TIMESTAMP[];

COMMIT;
