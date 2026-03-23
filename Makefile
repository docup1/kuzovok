# Makefile для Кузовок

BINARY_NAME ?= server
GO := go
GOFMT := gofmt
PHP ?= php

LOCAL_ADDR := 127.0.0.1:8080
PROXY_ADDR := 127.0.0.1:8090

GOOS ?= freebsd
GOARCH ?= amd64
CGO_ENABLED := 0

.PHONY: all build clean fmt run run-proxy run-prod test

all: build

fmt:
	@echo "Форматирование Go-кода..."
	find . -name "*.go" -not -path "./vendor/*" -exec $(GOFMT) -w {} +

test:
	@echo "Запуск тестов..."
	$(GO) test ./...

build:
	@echo "Сборка..."
	$(GO) build -o $(BINARY_NAME) ./cmd/server
	@echo "Готово: $(BINARY_NAME)"

run:
	@echo "Backend на http://$(LOCAL_ADDR)"
	$(GO) run ./cmd/server

run-proxy:
	@echo "PHP прокси на http://$(PROXY_ADDR)"
	KUSOVOK_BACKEND_URL=http://$(LOCAL_ADDR) $(PHP) -S $(PROXY_ADDR) index.php

run-prod:
	@echo "Остановка $(BINARY_NAME)..."
	-pkill -x $(BINARY_NAME)
	@echo "Запуск $(BINARY_NAME)..."
	nohup ./$(BINARY_NAME) > $(BINARY_NAME).log 2>&1 &

clean:
	@echo "Очистка..."
	rm -f $(BINARY_NAME)
	@echo "Очищено"
