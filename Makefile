# constants 
BINARY_NAME=dsc
CLI_PACKAGE=./cmd
TEST_PACKAGE=./...
VERSION=$(DASH_VERSION)

all: test release
# release project as .pkg or .tar.gz
release: release-darwin-aarch64 release-linux-x86_64
# builds compiler as executable
build: build-darwin-aarch64 build-linux-x86_64

test:
	# run compiler tests
	go test -v $(TEST_PACKAGE)

release-darwin-aarch64:
	$(MAKE) build-darwin-aarch64
	mkdir -p release
	pkgbuild --root . --scripts scripts/macos --install-location /usr/local/dash --identifier org.dash-lang.pkg.dash --version $(VERSION) release/dash-$(VERSION)-darwin-aarch64.pkg

release-linux-x86_64:
	$(MAKE) build-linux-x86_64
	mkdir -p release
	tar -zcvf release/dash-$(VERSION)-linux-x86_64.tar.gz --exclude='./release' --exclude='./.git' . 
	
build-darwin-aarch64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -X main.osName=darwin -X main.archName=aarch64" -o bin/$(BINARY_NAME) -v $(CLI_PACKAGE)  

build-linux-x86_64:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -X main.osName=linux -X main.archName=x86_64" -o bin/$(BINARY_NAME) -v $(CLI_PACKAGE)  
	
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -rf release/

	
