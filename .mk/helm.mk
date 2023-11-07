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

.PHONY: helm
helm: ## build the helm chart to a local archive, using ko for the image build
	cd deployment/helm; \
	    ko resolve --platform=${KO_PLATFORMS} --base-import-paths --push=${KO_PUSH_IMAGE} -f values.yaml > values.tmp.yaml && \
		mv values.tmp.yaml values.yaml && \
		helm dependency update && \
		helm package --version="${HELM_PACKAGE_VERSION}" . && \
		cat values.yaml
	git checkout deployment/helm/values.yaml

.PHONY: helm-template
helm-template: ## renders the helm templates which is useful for debugging
	cd deployment/helm; \
		helm dependency update && \
		helm template .
