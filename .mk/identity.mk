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

.PHONY: github-login password-login
github-login: ## setup GitHub login on Keycloak
ifndef KC_GITHUB_CLIENT_ID
	$(error KC_GITHUB_CLIENT_ID is not set)
endif
ifndef KC_GITHUB_CLIENT_SECRET
	$(error KC_GITHUB_CLIENT_SECRET is not set)
endif
	@echo "Setting up GitHub login..."
# Delete the existing GitHub identity provider, if it exists.  Otherwise, ignore the error.
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh delete identity-provider/instances/github -r stacklok || true
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances -r stacklok -s alias=github -s providerId=github -s enabled=true  -s 'config.useJwksUrl="true"' -s config.clientId=$$KC_GITHUB_CLIENT_ID -s config.clientSecret=$$KC_GITHUB_CLIENT_SECRET
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances/github/mappers -r stacklok -s name=gh_id -s identityProviderAlias=github -s identityProviderMapper=github-user-attribute-mapper -s config='{"syncMode":"FORCE", "jsonField":"id", "userAttribute":"gh_id"}'
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances/github/mappers -r stacklok -s name=gh_login -s identityProviderAlias=github -s identityProviderMapper=github-user-attribute-mapper -s config='{"syncMode":"FORCE", "jsonField":"login", "userAttribute":"gh_login"}'

password-login:
	@echo "Setting up password login..."
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create users -r stacklok -s username=testuser -s enabled=true
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh set-password -r stacklok --username testuser --new-password tester