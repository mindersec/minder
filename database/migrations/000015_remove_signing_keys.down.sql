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

-- signing_keys table
CREATE TABLE signing_keys (
                              id SERIAL PRIMARY KEY,
                              project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
                              private_key TEXT NOT NULL,
                              public_key TEXT NOT NULL,
                              passphrase TEXT NOT NULL,
                              key_identifier TEXT NOT NULL,
                              created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                              updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);