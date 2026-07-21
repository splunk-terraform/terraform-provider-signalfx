TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
WEBSITE_REPO=github.com/hashicorp/terraform-website
PKG_NAME=signalfx

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

test-with-cover:
	go test --race --timeout 300s ./... \
		-covermode=atomic \
		-coverprofile=$(PWD)/coverage.txt


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

.PHONY: build test test-with-cover testacc vet fmt fmtcheck errcheck gen-docs check-docs test-docs check-schema-docs
