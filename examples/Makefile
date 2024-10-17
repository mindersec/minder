# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0


## use absolute makefile location
MAKEFILE_DIR_LOCATION := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: clean
clean:: ## clean up environment
	rm -rf $(MAKEFILE_DIR_LOCATION)rules-and-profiles

.PHONY: init-examples
init-examples: clean ## clone example rules and profiles
	git clone https://github.com/mindersec/minder-rules-and-profiles.git $(MAKEFILE_DIR_LOCATION)rules-and-profiles || true
