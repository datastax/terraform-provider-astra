GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
PROVIDER=astra
INSTALL_PATH=~/.local/share/terraform/plugins/localhost/providers/$(PROVIDER)/0.0.1/linux_$(GOARCH)

ifeq ($(GOOS), darwin)
	INSTALL_PATH=~/Library/Application\ Support/io.terraform/plugins/localhost/providers/$(PROVIDER)/0.0.1/darwin_$(GOARCH)
endif
ifeq ($(GOOS), "windows")
	INSTALL_PATH=%APPDATA%/HashiCorp/Terraform/plugins/localhost/providers/$(PROVIDER)/0.0.1/windows_$(GOARCH)
endif

default: dev

dev:
	mkdir -p $(INSTALL_PATH)
	go build -o $(INSTALL_PATH)/terraform-provider-$(PROVIDER) main.go

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: dev testacc