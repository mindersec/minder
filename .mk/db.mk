# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: migrateup
migrateup: ## run migrate up
	@go run -tags '$(BUILDTAGS)' cmd/server/main.go migrate up --yes

.PHONY: migratedown
migratedown: ## run migrate down
	@go run -tags '$(BUILDTAGS)' cmd/server/main.go migrate down

.PHONY: dbschema
dbschema: ## generate database schema with schema spy, monitor file until doc is created and copy it
	mkdir -p database/schema/output && chmod a+w database/schema/output
	cd database/schema && KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko resolve --base-import-paths \
		--platform linux/$(DOCKERARCH) -f docker-compose.yaml | sed 's/^--*$$//' > .resolved-compose.yaml
	cd database/schema && $(COMPOSE) -f .resolved-compose.yaml run -u 1001:1001 --rm schemaspy \
		-configFile /config/schemaspy.properties -imageformat png
	sleep 10
	cp database/schema/output/diagrams/summary/relationships.real.large.png docs/static/img/minder/schema.png
	cd database/schema && $(COMPOSE) -f .resolved-compose.yaml down -v && rm -rf output .resolved-compose.yaml
