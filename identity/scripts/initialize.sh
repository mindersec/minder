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

# Wait for Keycloak to start and authenticate with admin credentials
while ! /opt/keycloak/bin/kcadm.sh config credentials --server http://keycloak:8080 --realm master --user "$KEYCLOAK_ADMIN" --password "$KEYCLOAK_ADMIN_PASSWORD"; do
  sleep 1
done

status=0
# Create realm stacklok, which stores account deletion events for 7 days
/opt/keycloak/bin/kcadm.sh get realms/stacklok >/dev/null 2>&1 || status="$?"; if [ $status -eq 1 ]; then
  /opt/keycloak/bin/kcadm.sh create realms -s realm=stacklok -s loginTheme=keycloak -s eventsEnabled=true -s 'enabledEventTypes=["DELETE_ACCOUNT"]' -s eventsExpiration=604800 -s enabled=true

  # Add account deletion capability to stacklok realm (see https://www.keycloak.org/docs/latest/server_admin/#authentication-operations)
  /opt/keycloak/bin/kcadm.sh update "/authentication/required-actions/delete_account" -r stacklok -b '{ "alias" : "delete_account", "name" : "Delete Account", "providerId" : "delete_account", "enabled" : true, "defaultAction" : false, "priority" : 60, "config" : { }}'

  # Give all users permission to delete their own account
  /opt/keycloak/bin/kcadm.sh add-roles -r stacklok --rname default-roles-stacklok --rolename delete-account --cclientid account
fi

# Create client scope which exposes the user's GitHub ID and login
/opt/keycloak/bin/kcadm.sh create client-scopes -r stacklok -f "$(dirname -- "${BASH_SOURCE[0]}")/scope.json"

# Create client minder-cli
if ! /opt/keycloak/bin/kcadm.sh get clients -r stacklok --fields 'clientId' | grep -q "minder-cli"; then
  /opt/keycloak/bin/kcadm.sh create clients -r stacklok -s clientId=minder-cli -s 'redirectUris=["http://localhost/*"]' -s publicClient=true -s enabled=true -s defaultClientScopes='["acr","email","profile","roles","web-origins","gh-data"]' -s optionalClientScopes='["microprofile-jwt","offline_access"]'
fi

# Create client minder-ui
if ! /opt/keycloak/bin/kcadm.sh get clients -r stacklok --fields 'clientId' | grep -q "minder-ui"; then
  /opt/keycloak/bin/kcadm.sh create clients -r stacklok -s clientId=minder-ui -s 'redirectUris=["http://localhost/*"]' -s publicClient=true -s enabled=true -s defaultClientScopes='["acr","email","profile","roles","web-origins","gh-data"]' -s optionalClientScopes='["microprofile-jwt","offline_access"]'
fi

# Create client minder-server to receive account deletion events
if ! /opt/keycloak/bin/kcadm.sh get clients -r stacklok --fields 'clientId' | grep -q "minder-server"; then
  /opt/keycloak/bin/kcadm.sh create clients -r stacklok -s clientId=minder-server -s serviceAccountsEnabled=true -s clientAuthenticatorType=client-secret -s secret="$KC_MINDER_SERVER_SECRET" -s enabled=true -s defaultClientScopes='["acr","email","profile","roles","web-origins","gh-data"]' -s optionalClientScopes='["microprofile-jwt","offline_access"]'

  # Give minder-server the capability to view events
  /opt/keycloak/bin/kcadm.sh add-roles -r stacklok --uusername service-account-minder-server --cclientid realm-management --rolename view-events

  # Give minder-server the capability to delete users
  /opt/keycloak/bin/kcadm.sh add-roles -r stacklok --uusername service-account-minder-server --cclientid realm-management --rolename manage-users
fi