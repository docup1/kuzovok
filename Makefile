# Makefile для Кузовок
# Сборка под FreeBSD

BINARY_NAME=kusovok
GOOS=freebsd
GOARCH=amd64
CGO_ENABLED=0

.PHONY: all build clean run freebsd

all: freebsd

freebsd:
	@echo "🔨 Сборка под FreeBSD (amd64)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o $(BINARY_NAME) .
	@echo "✅ Готово: $(BINARY_NAME)"

build:
	@echo "🔨 Сборка для текущей ОС..."
	go build -o $(BINARY_NAME) .
	@echo "✅ Готово: $(BINARY_NAME)"

clean:
	@echo "🧹 Очистка..."
	rm -f $(BINARY_NAME)
	@echo "✅ Очищено"

run: build
	@echo "🚀 Запуск $(BINARY_NAME)..."
	./$(BINARY_NAME)
