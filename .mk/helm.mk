# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: helm
helm: ## build the helm chart to a local archive, using ko for the image build
	cd deployment/helm; \
	    ko resolve --platform=${KO_PLATFORMS} --base-import-paths --push=${KO_PUSH_IMAGE} --image-refs built-images.yaml -f values.yaml > values.tmp.yaml && \
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

.PHONY: helm-docs
helm-docs: ## generate the helm docs
	cd deployment/helm; \
		helm-docs -t README.md.gotmpl -o README.md
