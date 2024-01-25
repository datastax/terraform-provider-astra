GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
PROVIDER=astra
DEV_VERSION=0.0.1
PLUGIN_PATH=registry.terraform.io/datastax/$(PROVIDER)/$(DEV_VERSION)/$(GOOS)_$(GOARCH)
TF_PLUGIN_DOCS_VERSION=v0.18.0


ifeq ($(GOOS), "windows")
        INSTALL_PATH=%APPDATA%/terraform.d/plugins/$(PLUGIN_PATH)
else
        INSTALL_PATH=~/.terraform.d/plugins/$(PLUGIN_PATH)
endif

default: build

build:
	mkdir -p bin
	go build -o bin/terraform-provider-$(PROVIDER)

install: build
	mkdir -p $(INSTALL_PATH)
	cp bin/terraform-provider-$(PROVIDER) $(INSTALL_PATH)

dev: install

test: testacc

testacc:
	test/run_tests.sh

clean:
	rm -f bin/terraform-provider-$(PROVIDER)

tools:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TF_PLUGIN_DOCS_VERSION)
	type terraform || (echo "'terraform' command not found, please install it" && exit 1)

docs: tools
	terraform fmt --recursive ./examples
	tfplugindocs

.PHONY: install build clean dev docs test testacc tools
