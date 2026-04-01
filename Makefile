APP_NAME := kosis
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BIN_DIR := bin
MAC_DIR := $(BIN_DIR)/mac
LINUX_DIR := $(BIN_DIR)/linux
WINDOWS_DIR := $(BIN_DIR)/windows

.PHONY: all build build-all clean test vet fmt check

# 기본: 로컬 빌드 (현재 OS/Arch)
all: build

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) .

# 크로스 컴파일 전체
# bin/
# ├── kosis              ← macOS arm64 (바로 실행용)
# ├── mac/
# │   └── kosis          ← macOS arm64
# ├── linux/
# │   └── kosis          ← Linux amd64
# └── windows/
#     └── kosis.exe      ← Windows amd64
build-all: clean
	@mkdir -p $(MAC_DIR) $(LINUX_DIR) $(WINDOWS_DIR)
	@echo "=== macOS (arm64) ==="
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(MAC_DIR)/$(APP_NAME) .
	cp $(MAC_DIR)/$(APP_NAME) $(BIN_DIR)/$(APP_NAME)
	@echo "=== Linux (amd64) ==="
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(LINUX_DIR)/$(APP_NAME) .
	@echo "=== Windows (amd64) ==="
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(WINDOWS_DIR)/$(APP_NAME).exe .
	@echo ""
	@echo "=== 빌드 완료 ==="
	@echo "bin/$(APP_NAME)              macOS arm64 (바로 실행)"
	@echo "bin/mac/$(APP_NAME)          macOS arm64"
	@echo "bin/linux/$(APP_NAME)        Linux amd64"
	@echo "bin/windows/$(APP_NAME).exe  Windows amd64"
	@echo ""
	@ls -lhR $(BIN_DIR)/

# 테스트
test:
	go test ./...

# 코드 점검
vet:
	go vet ./...

fmt:
	gofmt -w .

# 전체 점검 (fmt → vet → test → build)
check: fmt vet test build
	@echo "전체 점검 완료"

# 정리
clean:
	rm -rf $(BIN_DIR) $(APP_NAME)

# 설치 (로컬)
install: build
	cp $(BIN_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)
	@echo "$(APP_NAME) 설치 완료: /usr/local/bin/$(APP_NAME)"
