# Makefile для локальной разработки и сборки Кузовка

BINARY_NAME ?= kusovok
GO ?= go
GOFMT ?= gofmt
PHP ?= php

LOCAL_ADDR ?= 127.0.0.1:8080
PROXY_ADDR ?= 127.0.0.1:8090

GOOS ?= freebsd
GOARCH ?= amd64
CGO_ENABLED ?= 0

.PHONY: all build clean deploy-prod fmt freebsd run run-backend run-proxy run-prod test

all: build

fmt:
	@echo "🧼 Форматирование Go-кода..."
	$(GOFMT) -w main.go main_test.go

test:
	@echo "🧪 Запуск локальных тестов..."
	$(GO) test ./...

build:
	@echo "🔨 Сборка для текущей ОС..."
	$(GO) build -o $(BINARY_NAME) .
	@echo "✅ Готово: $(BINARY_NAME)"

run: run-backend

run-backend:
	@echo "🚀 Go backend на http://$(LOCAL_ADDR)"
	KUSOVOK_ADDR=$(LOCAL_ADDR) KUSOVOK_SECURE_COOKIE=false $(GO) run .

run-proxy:
	@echo "🌐 PHP прокси на http://$(PROXY_ADDR)"
	KUSOVOK_BACKEND_URL=http://$(LOCAL_ADDR) $(PHP) -S $(PROXY_ADDR) index.php

run-prod:
	@echo "🛑 Остановка запущенного ./$(BINARY_NAME), если он уже работает..."
	-pkill -x $(BINARY_NAME)
	@echo "🚀 Запуск ./$(BINARY_NAME) в фоне, лог: $(BINARY_NAME).log"
	nohup ./$(BINARY_NAME) > $(BINARY_NAME).log 2>&1 &

deploy-prod: build run-prod

freebsd:
	@echo "🔨 Сборка под FreeBSD ($(GOARCH))..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY_NAME) .
	@echo "✅ Готово: $(BINARY_NAME)"

clean:
	@echo "🧹 Очистка артефактов..."
	rm -f $(BINARY_NAME)
	@echo "✅ Очищено"
