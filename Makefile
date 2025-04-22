# 数据目录
DATA_DIR ?= $(HOME)/.frp-daemon

# 构建目录
BIN_DIR := build

# 目标文件
TARGETS := $(BIN_DIR)/fdctl $(BIN_DIR)/fdclient

.PHONY: all build install clean

all: build

build: $(TARGETS)

$(BIN_DIR)/fdctl: cmd/controller/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $@ $<

$(BIN_DIR)/fdclient: cmd/client/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $@ $<

install: build
	@mkdir -p /usr/local/bin
	cp $(BIN_DIR)/fdctl /usr/local/bin/
	cp $(BIN_DIR)/fdclient /usr/local/bin/
	chmod +x /usr/local/bin/fdctl
	chmod +x /usr/local/bin/fdclient

clean:
	rm -rf $(BIN_DIR) 