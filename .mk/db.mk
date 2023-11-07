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

.PHONY: migrateup
migrateup: ## run migrate up
	@go run -tags '$(BUILDTAGS)' cmd/server/main.go migrate up --yes

.PHONY: migratedown
migratedown: ## run migrate down
	@go run -tags '$(BUILDTAGS)' cmd/server/main.go migrate down

.PHONY: dbschema
dbschema: ## generate database schema with schema spy, monitor file until doc is created and copy it
	mkdir -p database/schema/output && chmod a+w database/schema/output
	cd database/schema && $(COMPOSE) run -u 1001:1001 --rm schemaspy -configFile /config/schemaspy.properties -imageformat png
	sleep 10
	cp database/schema/output/diagrams/summary/relationships.real.large.png docs/static/img/minder/schema.png
	cd database/schema && $(COMPOSE) down -v && rm -rf output
