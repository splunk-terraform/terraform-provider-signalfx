TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
WEBSITE_REPO=github.com/hashicorp/terraform-website
PKG_NAME=signalfx
COVERAGE_MIN ?= 90.0
COVERAGE_BASELINE ?= 34.7
MIGRATED_PRODUCT_PACKAGES := \
	./internal/framework/integration
FRAMEWORK_SCOPE_PACKAGES := \
	./internal/check \
	./internal/common \
	./internal/convert \
	./internal/feature \
	./internal/framework/... \
	./internal/providermeta \
	./internal/track \
	./internal/visual

SRC_ROOT        := $(shell git rev-parse --show-toplevel)
SRC_GO_FILES    := $(shell find $(SRC_ROOT) -name '*.go')
GOTOOLS_MOD_FILE   := $(SRC_ROOT)/internal/tools/go.mod
GOTOOLS_CMD        := go tool -modfile=$(GOTOOLS_MOD_FILE) 

ADDLICENCESE   := $(GOTOOLS_CMD) addlicense
GOVULNCHECK    := $(GOTOOLS_CMD) govulncheck
GOLANGCI_LINT  := $(GOTOOLS_CMD) golangci-lint
WEBSITE_PLUGIN := $(GOTOOLS_CMD) tfplugindocs

default: build

.PHONY: tool-cache
tool-cache:
	go -C $(shell dirname $(GOTOOLS_MOD_FILE)) mod download

.PHONY: addlicense
addlicense: tool-cache
	@ADDLICENCESEOUT=`$(ADDLICENCESE) -y "" -c 'Splunk, Inc.' -l mpl -s=only $(SRC_GO_FILES) 2>&1`; \
		if [ "$$ADDLICENCESEOUT" ]; then \
			echo "$(ADDLICENCESE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENCESEOUT\n"; \
			exit 1; \
		else \
			echo "Add License finished successfully"; \
		fi

.PHONY: checklicense
checklicense: tool-cache
	@ADDLICENCESEOUT=`$(ADDLICENCESE) -check $(SRC_GO_FILES) 2>&1`; \
		if [ "$$ADDLICENCESEOUT" ]; then \
			echo "$(ADDLICENCESE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENCESEOUT\n"; \
			echo "Use 'make addlicense' to fix this."; \
			exit 1; \
		else \
			echo "Check License finished successfully"; \
		fi

.PHONY: govulncheck
govulncheck: tool-cache
	$(GOVULNCHECK) ./...

.PHONY: lint
lint: tool-cache
	$(GOLANGCI_LINT) run -v

.PHONY: lint-fix
lint-fix: tool-cache
	$(GOLANGCI_LINT) run -v --fix

build:
	go build

test:
	go test --cover --race -v --timeout 30s ./...

# Framework Protocol 6 lifecycle tests start local Terraform/provider processes.
# The cumulative integration package exceeds five minutes on shared CI runners.
test-with-cover:
	go test --race --timeout 600s ./... \
		-covermode=atomic \
		-coverprofile=$(PWD)/coverage.txt

test-migrated-product-cover:
	mkdir -p $(PWD)/coverage/products
	@set -eu; \
		for package in $(MIGRATED_PRODUCT_PACKAGES); do \
			name=$${package##*/}; \
			profile=$(PWD)/coverage/products/$$name.txt; \
			go test --timeout 300s $$package \
				-covermode=atomic \
				-coverprofile=$$profile; \
			total=$$(go tool cover -func=$$profile | awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }'); \
			echo "Migrated product package $$package coverage: $$total% (minimum $(COVERAGE_MIN)%)"; \
			awk -v actual="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; \
		done

# This aggregate target is the final migration gate. Incremental checkpoints use
# test-migrated-product-cover so existing Framework packages do not make every
# one-resource review branch fail before their planned coverage closure.
test-framework-cover:
	mkdir -p $(PWD)/coverage
	go test --timeout 300s $(FRAMEWORK_SCOPE_PACKAGES) \
		-covermode=atomic \
		-coverprofile=$(PWD)/coverage/framework.txt
	@total=$$(go tool cover -func=$(PWD)/coverage/framework.txt | awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }'); \
		echo "Non-deferred Framework/core coverage: $$total% (minimum $(COVERAGE_MIN)%)"; \
		awk -v actual="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'

check-coverage-no-regression: test-with-cover
	@total=$$(go tool cover -func=$(PWD)/coverage.txt | awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }'); \
		echo "Full-module coverage: $$total% (baseline $(COVERAGE_BASELINE)%)"; \
		awk -v actual="$$total" -v minimum="$(COVERAGE_BASELINE)" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'


testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

fmt:
	gofmt -w $(GOFMT_FILES)


test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

check-docs: gen-docs
	@if [ "`git status --porcelain docs/`" ];then \
		git diff;\
		echo "Changes to documentation are not committed. Please run 'make gen-docs' and commit the changes" && \
		echo `git status --porcelain docs/` &&\
		exit 1;\
	fi


gen-docs:
	$(WEBSITE_PLUGIN)

test-docs:
	$(WEBSITE_PLUGIN) validate 

check-schema-docs:
	./scripts/check-schema-docs.sh

.PHONY: build test test-with-cover test-migrated-product-cover test-framework-cover check-coverage-no-regression testacc vet fmt fmtcheck errcheck gen-docs check-docs test-docs check-schema-docs
