# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

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
# TODO(evankanderson): Move to a non-branded realm name for development...
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh delete identity-provider/instances/github -r stacklok || true
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances -r stacklok -s alias=github -s providerId=github -s enabled=true  -s 'config.useJwksUrl="true"' -s config.clientId=$$KC_GITHUB_CLIENT_ID -s config.clientSecret=$$KC_GITHUB_CLIENT_SECRET
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances/github/mappers -r stacklok -s name=gh_id -s identityProviderAlias=github -s identityProviderMapper=github-user-attribute-mapper -s config='{"syncMode":"FORCE", "jsonField":"id", "userAttribute":"gh_id"}'
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances/github/mappers -r stacklok -s name=gh_login -s identityProviderAlias=github -s identityProviderMapper=github-user-attribute-mapper -s config='{"syncMode":"FORCE", "jsonField":"login", "userAttribute":"gh_login"}'

password-login:
	@echo "Setting up password login..."
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create users -r stacklok -s username=testuser -s enabled=true -s email=testuser@example.com -s 'attributes.Demo=["value"]'
	@$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh set-password -r stacklok --username testuser --new-password tester
