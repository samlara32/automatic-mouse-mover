COVER_PROFILE=cover.out
COVER_HTML=cover.html

SIGN_IDENTITY ?= $(shell security find-identity -v -p codesigning 2>/dev/null | grep -q "AMM Code Signing" && echo "AMM Code Signing" || echo "-")

.PHONY: all build build-arm64 build-amd64 package install uninstall release clean start test coverage vet lint $(COVER_PROFILE) $(COVER_HTML)

all: package

build: clean
	mkdir -p -v ./bin/amm.app/Contents/Resources/assets/icon
	mkdir -p -v ./bin/amm.app/Contents/MacOS
	cp ./appInfo/*.plist ./bin/amm.app/Contents/Info.plist
	cp ./appInfo/*.icns ./bin/amm.app/Contents/Resources/icon.icns
	cp ./assets/icon/* ./bin/amm.app/Contents/Resources/assets/icon
	CGO_ENABLED=1 GOARCH=arm64 go build -o ./bin/amm_arm64 cmd/main.go
	CGO_ENABLED=1 GOARCH=amd64 CC="clang -target x86_64-apple-macos10.13" go build -o ./bin/amm_amd64 cmd/main.go
	lipo -create -output ./bin/amm.app/Contents/MacOS/amm ./bin/amm_arm64 ./bin/amm_amd64
	rm -f ./bin/amm_arm64 ./bin/amm_amd64
	@if [ "$(SIGN_IDENTITY)" = "AMM Code Signing" ]; then \
		echo "Signing with persistent AMM Code Signing identity..."; \
		codesign --force --deep -s "AMM Code Signing" ./bin/amm.app; \
	else \
		codesign --force --deep -s - -i "com.pg.amm" --requirements '=designated => identifier "com.pg.amm"' ./bin/amm.app; \
	fi

build-arm64: clean
	mkdir -p -v ./bin/amm.app/Contents/Resources/assets/icon
	mkdir -p -v ./bin/amm.app/Contents/MacOS
	cp ./appInfo/*.plist ./bin/amm.app/Contents/Info.plist
	cp ./appInfo/*.icns ./bin/amm.app/Contents/Resources/icon.icns
	cp ./assets/icon/* ./bin/amm.app/Contents/Resources/assets/icon
	CGO_ENABLED=1 GOARCH=arm64 go build -o ./bin/amm.app/Contents/MacOS/amm cmd/main.go
	@if [ "$(SIGN_IDENTITY)" = "AMM Code Signing" ]; then \
		echo "Signing with persistent AMM Code Signing identity..."; \
		codesign --force --deep -s "AMM Code Signing" ./bin/amm.app; \
	else \
		codesign --force --deep -s - -i "com.pg.amm" --requirements '=designated => identifier "com.pg.amm"' ./bin/amm.app; \
	fi

package: build
	rm -f ./bin/AutomaticMouseMover.dmg ./bin/Applications
	ln -s /Applications ./bin/Applications
	hdiutil create -volname "Automatic Mouse Mover" -srcfolder ./bin/amm.app -ov -format UDZO AutomaticMouseMover.dmg
	rm -f ./bin/Applications
	mv AutomaticMouseMover.dmg ./bin/

install: build
	killall amm 2>/dev/null || true
	rm -rf /Applications/amm.app
	cp -R ./bin/amm.app /Applications/amm.app
	xattr -cr /Applications/amm.app

uninstall:
	killall amm 2>/dev/null || true
	tccutil reset Accessibility com.pg.amm 2>/dev/null || true
	tccutil reset Accessibility /Applications/amm.app 2>/dev/null || true
	rm -rf /Applications/amm.app
	rm -rf ~/.config/amm
	@echo "AMM has been completely uninstalled and Accessibility permissions removed."

release:
	@./scripts/release.sh

clean:
	rm -rf ./bin $(COVER_PROFILE) $(COVER_HTML)

start:
	go run cmd/main.go

test: coverage

coverage: $(COVER_HTML)

$(COVER_HTML): $(COVER_PROFILE)
	go tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML)

$(COVER_PROFILE):
	go test -v -failfast -race -coverprofile=$(COVER_PROFILE) ./...

lint:
	go vet ./...