# Кузовок

Социальная сеть с Go backend на DDD-архитектуре, SQLite-хранилищем и статическим frontend.

## Архитектура (DDD)

```
internal/
├── domain/           # Сущности, интерфейсы репозиториев, доменные сервисы
│   ├── user/         # Пользователи
│   ├── post/         # Посты
│   ├── like/         # Лайки
│   ├── access/       # Управление доступом
│   └── shared/       # Общие типы (Role, UserSummary, AccessInfo)
├── application/      # Use cases (сценарии использования)
│   ├── authapp/      # Регистрация, логин, /me
│   ├── postapp/      # Создание постов, лента
│   ├── likeapp/      # Лайк/unlike
│   └── admin/        # Админские операции
├── infrastructure/  # Реализации (БД, auth, storage)
│   ├── config/       # Загрузка config.json
│   ├── database/     # SQLite репозитории
│   ├── auth/         # JWT, cookies
│   └── storage/      # Хранение картинок
└── handlers/         # HTTP обработчики, middleware, router

cmd/server/main.go    # Точка входа
config.json           # Конфигурация
static/               # Фронтенд
img/                  # Временные картинки
```

## Конфигурация

Все настройки в `config.json`:

```json
{
  "server": { "addr": ":8080" },
  "database": { "path": "./kusovok.db", "max_open_conns": 25 },
  "auth": {
    "jwt_secret": "change-me-in-production",
    "jwt_expire_hours": 24,
    "cookie_name": "token",
    "cookie_path": "/",
    "secure_cookie": false
  },
  "images": {
    "dir": "./img",
    "public_path": "/img/",
    "lifetime_hours": 24,
    "max_size_mb": 10
  },
  "messages": {
    "access_denied": "извините, вы пока не кузовок",
    "admin_denied": "доступ только для администратора"
  },
  "limits": {
    "post_content_max_length": 5000
  },
  "cleanup": {
    "interval_minutes": 60
  }
}
```

## Запуск

```bash
# Сборка
make build

# Запуск
make run

# Или напрямую
go run cmd/server/main.go
```

Приложение доступно на [http://localhost:8080](http://localhost:8080).

## API

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /api/register | Регистрация |
| POST | /api/login | Логин |
| POST | /api/logout | Выход |
| GET | /api/me | Текущий пользователь |
| GET | /api/feed | Лента постов |
| POST | /api/posts | Создать пост |
| GET | /api/posts | Посты пользователя |
| POST | /api/like | Лайк/анлайк |
| GET | /api/admin/users | Все пользователи |
| GET | /api/admin/likes | Лайки постов |
| GET | /api/admin/allowed-users | Allowlist |
| POST | /api/admin/allowed-users | Добавить в allowlist |
| PUT | /api/admin/allowed-users/{id}/role | Изменить роль |
| DELETE | /api/admin/allowed-users/{id} | Удалить из allowlist |

## Allowlist-доступ

По умолчанию allowlist пуст. Новые пользователи видят сообщение "извините, вы пока не кузовок".

```sql
-- Обычный доступ
INSERT INTO allowed_users (user_id) VALUES (1);

-- Первый админ
INSERT INTO allowed_users (user_id, role) VALUES (1, 'admin');
```

## Команды Makefile

```bash
make build      # Сборка
make run        # Запуск
make test       # Тесты
make fmt        # Форматирование
make clean      # Очистка
```

## Продовый деплой

Для shared-хостинга используются:

- `.htaccess` — маршрутизация под путь вида `/~user/kuzovok/`
- `index.php` — проксирование `/api/*` в локальный Go backend и раздача статики
