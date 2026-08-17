COVER_PROFILE=cover.out
COVER_HTML=cover.html

.PHONY: all build build-arm64 build-amd64 package install clean start test coverage vet lint $(COVER_PROFILE) $(COVER_HTML)

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
	codesign --force --deep --sign - ./bin/amm.app

build-arm64: clean
	mkdir -p -v ./bin/amm.app/Contents/Resources/assets/icon
	mkdir -p -v ./bin/amm.app/Contents/MacOS
	cp ./appInfo/*.plist ./bin/amm.app/Contents/Info.plist
	cp ./appInfo/*.icns ./bin/amm.app/Contents/Resources/icon.icns
	cp ./assets/icon/* ./bin/amm.app/Contents/Resources/assets/icon
	CGO_ENABLED=1 GOARCH=arm64 go build -o ./bin/amm.app/Contents/MacOS/amm cmd/main.go
	codesign --force --deep --sign - ./bin/amm.app

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