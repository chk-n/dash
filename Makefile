# constants 
BINARY_NAME=dash
CLI_PACKAGE=./cmd
TEST_PACKAGE=./...
VERSION=$(DASH_VERSION)

all: test release
# release project as .pkg or .tar.gz
release: release-macos-aarch64 release-linux-aarch64
# builds compiler as executable
build: build-macos-aarch64 build-linux-aarch64

test:
	# run compiler tests
	go test -v $(TEST_PACKAGE)
	# run end-to-end tests (requires dash to be installed)
	cd tests/ && lua run_tests.lua

release-macos-aarch64:
	$(MAKE) build-darwin-aarch64
	mkdir release
	pkgbuild --root . --scripts scripts/macos --install-location /usr/local/dash --identifier org.dash-lang.pkg.dash --version $(VERSION) release/dash-$(VERSION)-macos-aarch64.pkg

release-linux-x86_64:
	$(MAKE) build-linux-x86_64
	mkdir release
	tar -zcvf release/dash-$(VERSION)-linux-x86_64.tar.gz --exclude='./release' --exclude='./.git' . 
	
build-darwin-aarch64:
	rm -rf macos-minimal-sdk
	git clone https://github.com/tinygo-org/macos-minimal-sdk
	cd macos-minimal-sdk && make CLANG=clang sysroot-macos-arm64 && cd ..
	mv macos-minimal-sdk/sysroot-macos-arm64/usr/lib/libSystem.B.dylib lib/libSystem-macos-arm64.dylib
	rm -rf macos-minimal-sdk
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -X main.osName=darwin -X main.archName=aarch64" -o bin/$(BINARY_NAME) -v $(CLI_PACKAGE)  

build-linux-x86_64:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -X main.osName=linux -X main.archName=aarch64" -o bin/$(BINARY_NAME) -v $(CLI_PACKAGE)  
	
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -rf release/

	
