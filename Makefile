NAME=vmware
BINARY=packer-plugin-${NAME}

COUNT?=1
TEST?=$(shell go list ./...)
HASHICORP_PACKER_PLUGIN_SDK_VERSION?=$(shell go list -m github.com/hashicorp/packer-plugin-sdk | cut -d " " -f2)

.PHONY: dev

build:
	@go build -o ${BINARY}

dev: build
	@mkdir -p ~/.packer.d/plugins/
	@mv ${BINARY} ~/.packer.d/plugins/${BINARY}

test:
	@go test -race -count $(COUNT) $(TEST) -timeout=3m

install-packer-sdc: ## Install packer sofware development command
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@${HASHICORP_PACKER_PLUGIN_SDK_VERSION}

ci-release-docs: install-packer-sdc
	@packer-sdc renderdocs -src docs -partials docs-partials/ -dst docs/
	@/bin/sh -c "[ -d docs ] && zip -r docs.zip docs/"

plugin-check: install-packer-sdc build
	@packer-sdc plugin-check ${BINARY}

testacc: dev
	@PACKER_ACC=1 go test -count $(COUNT) -v $(TEST) -timeout=120m

generate: install-packer-sdc
	@go generate ./...
	packer-sdc renderdocs -src ./docs -dst ./.docs -partials ./docs-partials
	# checkout the .docs folder for a preview of the docs

build-docs: install-packer-sdc
	@if [ -d ".docs" ]; then rm -r ".docs"; fi
	@packer-sdc renderdocs -src "docs" -partials docs-partials/ -dst ".docs/"
	@./.web-docs/scripts/compile-to-webdocs.sh "." ".docs" ".web-docs" "hashicorp"
	@rm -r ".docs"


### hack: release windows amd64

export GH_REPO = rstms/$(BINARY)
LABEL = x5.0_windows_amd64
RELEASE != git tag -l --sort v:refname | tail -1
GITHUB_RELEASE != gh release view --json tagName --jq .tagName
DOTEXE = $(BINARY)_$(RELEASE)_$(LABEL).exe
ZIPFILE = $(BINARY)_$(RELEASE)_$(LABEL).zip
CHECKSUMS = $(BINARY)_$(RELEASE)_SHA256SUMS

$(DOTEXE): $(BINARY)
	cp $< $@

$(ZIPFILE): $(DOTEXE)
	zip $@ $<

$(CHECKSUMS): $(ZIPFILE)
	sha256sum $< | tr -d '*' >$@

github_release:
	$(if $(filter $(GITHUB_RELEASE),$(RELEASE)),,gh release create $(RELEASE) --notes 'Release $(RELEASE)')
	@:

.zip_uploaded: $(ZIPFILE) github_release
	gh release upload --clobber $(RELEASE) $(ZIPFILE)
	@touch $@

.checksums_uploaded: $(CHECKSUMS) .zip_uploaded
	gh release upload --clobber $(RELEASE) $(CHECKSUMS)
	@touch $@

release: .checksums_uploaded

bump:
	bumpversion patch && git push --tags

clean:
	rm -f $(BINARY) *.exe *.zip *SHA256SUMS .*_uploaded

hidden_vars := hidden_vars .DEFAULT_GOAL CURDIR MAKEFILE_LIST MAKEFLAGS SHELL
showvars:
	@:;$(foreach var,$(filter-out $(hidden_vars),$(sort $(.VARIABLES))),$(if $(filter file%,$(origin $(var))),$(info $(var)=$($(var))),))


