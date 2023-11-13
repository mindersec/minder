-- Copyright 2023 Stacklok, Inc
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

-- see bug #1608: we were not linking PRs properly with rule evaluations. Just drop those rows, there's nothing
-- we can do with them.
DELETE FROM rule_evaluations WHERE entity = 'pull_request' AND pull_request_id IS NULL;

-- transaction commit
COMMIT;
