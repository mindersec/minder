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

-- external_id is a serialization of a protobuf struct which contains
-- provider-specific used to identify the entity
-- e.g. a github repo may contain info on owner and repo name, or an OCI
-- provider may contain the image name/tag and digest
-- TODO: make these non-nullable once we populate the table
ALTER TABLE repositories ADD COLUMN external_id BYTEA;
ALTER TABLE artifacts ADD COLUMN external_id BYTEA;
ALTER TABLE pull_requests ADD COLUMN external_id BYTEA;
