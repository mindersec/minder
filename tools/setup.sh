#!/bin/bash

#
# Copyright 2023 Stacklok, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eux

medic() {
  go run ./cmd/cli/main.go "$@";
}
# use root:root for credentials
medic auth login

medic provider enroll -n github

echo '$ medic rule_type create -f examples/github/rule-types/'
echo '---'
medic rule_type create -f examples/github/rule-types/

#echo '$ medic profile create -f examples/github/profiles/profile.yaml'
#echo '---'
medic profile create -f examples/github/profiles/profile.yaml
