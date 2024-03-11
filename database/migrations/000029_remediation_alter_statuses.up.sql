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

-- Add pending and unknown to the remediation status types enum
ALTER TYPE remediation_status_types add value 'pending';
ALTER TYPE remediation_status_types add value 'unknown';

-- Add pending and unknown to the alert status types enum
ALTER TYPE alert_status_types add value 'pending';
ALTER TYPE alert_status_types add value 'unknown';
