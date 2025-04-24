# 数据目录
DATA_DIR ?= $(HOME)/.frp-daemon

# 构建目录
BIN_DIR := build

# 目标平台
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# 默认目标
.PHONY: all build install clean cross-build

all: build

# 本地构建
build: $(BIN_DIR)/fdctl $(BIN_DIR)/fdclient

$(BIN_DIR)/fdctl: cmd/fdctl/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $@ $<

$(BIN_DIR)/fdclient: cmd/fdclient/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $@ $<

# 交叉编译
cross-build:
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then \
		echo "Usage: make cross-build GOOS=<os> GOARCH=<arch>"; \
		echo "Available platforms:"; \
		for platform in $(PLATFORMS); do \
			echo "  $$platform"; \
		done; \
		exit 1; \
	fi
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BIN_DIR)/fdctl-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,) cmd/fdctl/main.go
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BIN_DIR)/fdclient-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,) cmd/fdclient/main.go

# 安装
install: build
	@mkdir -p /usr/local/bin
	cp $(BIN_DIR)/fdctl /usr/local/bin/
	cp $(BIN_DIR)/fdclient /usr/local/bin/
	chmod +x /usr/local/bin/fdctl
	chmod +x /usr/local/bin/fdclient

# 清理
clean:
	rm -rf $(BIN_DIR) 