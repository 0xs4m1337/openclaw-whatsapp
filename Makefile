APP      := openclaw-whatsapp
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build release install clean install-service

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(APP) .

release:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="dist/$(APP)-$$os-$$arch$$ext"; \
		echo "  build $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags="$(LDFLAGS)" -o "$$out" . || exit 1; \
	done

install: build
	install -d /usr/local/bin
	install -m 755 $(APP) /usr/local/bin/$(APP)

clean:
	rm -f $(APP)
	rm -rf dist

install-service:
	mkdir -p $(HOME)/.config/systemd/user
	cp openclaw-whatsapp.service $(HOME)/.config/systemd/user/
	systemctl --user daemon-reload
	systemctl --user enable openclaw-whatsapp
	@echo "Start with: systemctl --user start openclaw-whatsapp"
