#!/usr/bin/env bash

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

set -euo pipefail

while ! /opt/keycloak/bin/kcadm.sh config credentials --server http://keycloak:8080 --realm master --user "$KEYCLOAK_ADMIN" --password "$KEYCLOAK_ADMIN_PASSWORD"; do
  sleep 1
done

/opt/keycloak/bin/kcadm.sh create realms -s realm=stacklok -s loginTheme=keycloak -s enabled=true
/opt/keycloak/bin/kcadm.sh create clients -r stacklok -s clientId=mediator-cli -s 'redirectUris=["http://localhost/*"]' -s publicClient=true -s enabled=true
/opt/keycloak/bin/kcadm.sh create clients -r stacklok -s clientId=mediator-ui -s 'redirectUris=["http://localhost/*"]' -s publicClient=true -s enabled=true
