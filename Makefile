PROJECTNAME=$(shell basename "$(PWD)")
VERSION=-ldflags="-X main.Version=$(shell git describe --tags)"

.PHONY: help run build install license
all: help

help: Makefile
	@echo
	@echo "Choose a make command to run in "$(PROJECTNAME)":"
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
	@echo

	@echo "Important targets:"
	@printf "  %-35s - %s\n" "import-core" "Do it before your first build"
	@printf "  %-35s - %s\n" "get" "Downloading & Installing all the modules"
	@printf "  %-35s - %s\n" "build" "Build the project"

check-git:
	@which git > /dev/null || (echo "git is not installed. Please install and try again."; exit 1)

check-go:
	@which go > /dev/null || (echo "Go is not installed.. Please install and try again."; exit 1)

check-protoc:
	@which protoc > /dev/null || (echo "protoc is not installed. Please install and try again."; exit 1)

protoc:
	protoc --go_out=. --go-grpc_out=. ./server/proto/*.proto
	protoc --go_out=. --go-grpc_out=. ./miner/proto/*.proto

get:
	@echo "  >  \033[32mDownloading & Installing all the modules...\033[0m "
	go mod tidy && go mod download

build-example: check-go check-git
	@echo "  >  \033[32mBuilding binary...\033[0m "
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval VERSION = $(shell git tag --points-at ${COMMIT_HASH}))
	go build -o build/webapp example/main.go

build: check-go check-git
	@echo "  >  \033[32mBuilding binary...\033[0m "
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval VERSION = $(shell git tag --points-at ${COMMIT_HASH}))
	go build -o build/edge-matrix-computing -ldflags="\
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Version=$(VERSION)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Branch=$(BRANCH)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Build=$(COMMIT_HASH)'"\
    main.go

build-linux-amd64: check-go check-git
	@echo "  >  \033[32mBuilding linux-amd64 binary...\033[0m "
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval VERSION = $(shell git tag --points-at ${COMMIT_HASH}))
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/edge-matrix-computing -ldflags="\
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Version=$(VERSION)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Branch=$(BRANCH)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Build=$(COMMIT_HASH)'"\
    main.go

build-linux-arm64: check-go check-git
	@echo "  >  \033[32mBuilding linux-arm64 binary...\033[0m "
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval VERSION = $(shell git tag --points-at ${COMMIT_HASH}))
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/linux-arm64/edge-matrix-computing -ldflags="\
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Version=$(VERSION)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Branch=$(BRANCH)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Build=$(COMMIT_HASH)'"\
    main.go

build-windows-amd64: check-go check-git
	@echo "  >  \033[32mBuilding windows-amd64 binary...\033[0m "
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval VERSION = $(shell git tag --points-at ${COMMIT_HASH}))
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/windows-amd64/edge-matrix-computing.exe -ldflags="\
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Version=$(VERSION)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Branch=$(BRANCH)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Build=$(COMMIT_HASH)'"\
    main.go

build-darwin-arm64: check-go check-git
	@echo "  >  \033[32mBuilding darwin-arm64 binary...\033[0m "
	$(eval COMMIT_HASH = $(shell git rev-parse HEAD))
	$(eval BRANCH = $(shell git rev-parse --abbrev-ref HEAD | tr -d '\040\011\012\015\n'))
	$(eval VERSION = $(shell git tag --points-at ${COMMIT_HASH}))
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o build/darwin-arm64/edge-matrix-computing -ldflags="\
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Version=$(VERSION)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Branch=$(BRANCH)' \
         -X 'github.com/EdgeMatrixChain/edge-matrix-computing/versioning.Build=$(COMMIT_HASH)'"\
    main.go

import-core:
	TARGET=build ./scripts/import_core_module.sh

reimport-core:
	rm -rf edge-matrix-core/
	TARGET=build ./scripts/import_core_module.sh

clean-build:
	rm -rf build/
