GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
PROVIDER=astra
DEV_VERSION=0.0.1
PLUGIN_PATH=registry.terraform.io/datastax/$(PROVIDER)/$(DEV_VERSION)/$(GOOS)_$(GOARCH)

ifeq ($(GOOS), "windows")
        INSTALL_PATH=%APPDATA%/terraform.d/plugins/$(PLUGIN_PATH)
else
        INSTALL_PATH=~/.terraform.d/plugins/$(PLUGIN_PATH)
endif

default: install

install: build
	mkdir -p $(INSTALL_PATH)
	cp bin/terraform-provider-$(PROVIDER) $(INSTALL_PATH)

build:
	mkdir -p bin
	go build -o bin/terraform-provider-$(PROVIDER)

dev: install

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

clean:
	rm bin/terraform-provider-$(PROVIDER)

.PHONY: install build clean dev testacc
