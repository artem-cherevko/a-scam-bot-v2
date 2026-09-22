# 🛡️ A-Scam Bot v2

Telegram-бот и HTTP API для проверки пользователей на скам, управления гарантиями, ролями и Trainee.

Проект построен на Go, PostgreSQL, GORM и Telegram Bot API. Бот и API работают в одном приложении и используют общую бизнес-логику через repository/service layers.

---

## ✨ Возможности

### 🔎 Проверка пользователей

- Поиск пользователя по Telegram ID.
- Проверка роли пользователя.
- Просмотр информации о пользователе.
- Работа со списком скамеров.
- Хранение причины добавления в список скамеров.
- Счётчики предупреждений и поисков.

### 🛡️ Гаранты

- Список гарантов.
- Обычные гаранты (`basic`).
- Топ-гаранты (`top`).
- Добавление гаранта по Telegram ID или username.
- Редактирование гаранта.
- Изменение ранга.
- Изменение канала.
- Изменение количества пруфов.
- Изменение региона.
- Сброс дополнительных данных гаранта.
- Хранение списка Trainee.

### 👥 Trainee

Гарант и топ-гарант могут:

- назначить пользователя Trainee;
- удалить пользователя из своих Trainee;
- автоматически изменять роль пользователя при назначении/удалении.

### 👮 Роли

Поддерживаются роли:

```text
tech
owner
co-owner
senior-admin
admin
junior-admin
intern
top-guarantor
guarantor
trainee
scammer
dodgy-character
user
```

### 🔐 API

HTTP API используется ботом для выполнения административных операций.

API содержит:

- работу с пользователями;
- работу с гарантами;
- назначение Trainee;
- удаление Trainee;
- административные операции;
- middleware проверки доступа.

---

# 🧱 Архитектура

Проект использует разделение на несколько уровней:

```text
Telegram Bot
     │
     ▼
  Handlers
     │
     ▼
  API Client
     │
     ▼
   HTTP API
     │
     ▼
  Middleware
     │
     ▼
   Services
     │
     ▼
 Repositories
     │
     ▼
 PostgreSQL
```

Основные компоненты:

```text
internal/
├── api/
│   ├── handlers
│   └── middleware
│
├── bot/
│   ├── handlers
│   ├── FSM
│   └── bot startup
│
├── database/
│   ├── models
│   ├── roles
│   └── migrations
│
├── repository/
│   ├── user repository
│   └── guarantor repository
│
├── service/
│   └── business logic
│
└── config/
    └── application configuration
```

---

# 🛠️ Стек

## Backend

- Go 1.27+
- Gin
- GORM
- PostgreSQL
- Telegram Bot API

## Infrastructure

- Docker
- Docker Compose
- GitHub Actions
- GitHub Container Registry (GHCR)

## Дополнительно

- FSM для многошаговых Telegram-команд
- REST API
- PostgreSQL constraints
- Role-based access control

---

# 📦 Требования

Для запуска локально:

- Go 1.27+
- PostgreSQL 18+
- Git
- Telegram Bot Token

Для запуска через Docker:

- Docker
- Docker Compose

---

# 🚀 Запуск локально

## 1. Клонирование

```bash
git clone https://github.com/artem-cherevko/a-scam-bot-v2.git
cd a-scam-bot-v2
```

---

## 2. Конфигурация

Создайте `.env`:

```env
BOT_TOKEN=your_telegram_bot_token

DB_DSN=postgres://ascam:password@localhost:5432/ascam?sslmode=disable

API_ENDPOINT=http://localhost:8080

PORT=8080
```

Названия дополнительных переменных зависят от используемой конфигурации приложения.

**Не добавляйте `.env` в Git.**

---

## 3. PostgreSQL

Создайте базу:

```sql
CREATE USER ascam WITH PASSWORD 'your_password';
CREATE DATABASE ascam OWNER ascam;
```

После этого укажите DSN:

```env
DB_DSN=postgres://ascam:your_password@localhost:5432/ascam?sslmode=disable
```

---

## 4. Запуск

```bash
go run .
```

Или, если entrypoint находится в `cmd`:

```bash
go run ./cmd
```

---

# 🐳 Docker

Проект может запускаться вместе с PostgreSQL.

Пример:

```bash
docker compose up -d
```

Проверить состояние:

```bash
docker compose ps
```

Посмотреть логи:

```bash
docker compose logs -f app
```

Остановить:

```bash
docker compose down
```

---

# 🗄️ PostgreSQL в Docker

Для подключения к PostgreSQL внутри контейнера:

```bash
docker exec -it a-scam-postgres psql -U ascam -d ascam
```

Проверка таблиц:

```sql
\dt
```

Основные таблицы:

```text
users
guarantors
```

---

# 🔧 Docker Compose

В Docker приложение обращается к PostgreSQL по имени сервиса:

```env
DB_DSN=postgres://ascam:PASSWORD@postgres:5432/ascam?sslmode=disable
```

Если API и Telegram-бот находятся в одном контейнере:

```env
API_ENDPOINT=http://localhost:8080
```

Если API находится в отдельном контейнере:

```env
API_ENDPOINT=http://app:8080
```

---

# 👮 Система ролей

## Tech

```text
tech
```

Техническая роль с максимальным административным уровнем.

## Owner

```text
owner
```

Владелец проекта.

## Co-Owner

```text
co-owner
```

Заместитель владельца.

## Senior Admin

```text
senior-admin
```

Старший администратор.

## Admin

```text
admin
```

Администратор.

## Junior Admin

```text
junior-admin
```

Младший администратор.

## Intern

```text
intern
```

Стажёр.

## Top Guarantor

```text
top-guarantor
```

Топ-гарант.

## Guarantor

```text
guarantor
```

Обычный гарант.

## Trainee

```text
trainee
```

Trainee гаранта.

## Scammer

```text
scammer
```

Пользователь, добавленный в список скамеров.

## Dodgy Character

```text
dodgy-character
```

Пользователь с соответствующим статусом проверки.

## Regular User

```text
user
```

Обычный пользователь.

---

# 🛡️ Команды и возможности бота

## 👤 Обычные команды

### `/start`

Регистрация пользователя в базе данных и вывод стартовой информации.

```text
/start
```

Если пользователь уже зарегистрирован, API обрабатывает повторный `/start` без создания дубликата.

---

### `/check <ID/@username>`

Проверить пользователя по Telegram ID или username.

```text
/check 8237096467
```

или:

```text
/check @username
```

В карточке отображаются:

- 👤 username;
- 🪪 Telegram ID;
- ⚙️ текущая роль;
- 🚨 причина для `scammer` / `dodgy-character`;
- ⚠️ количество предупреждений для административных ролей;
- 🚨 количество добавленных скамеров;
- 🔎 количество проверок.

Если пользователя нет в базе, бот всё равно показывает карточку с пометкой `Нет в базе`.

---

### `/me`

Показать информацию о собственном аккаунте.

```text
/me
```

---

# 👮 Административные команды

## 🔴 Уровни доступа

В коде используются две основные проверки:

### `canSetRole`

Доступ к изменению ролей, `/add_g`, `/warn` и `/unwarn` имеют:

```text
tech
owner
co-owner
senior-admin
```

### `isAdminRole`

Административный доступ имеют:

```text
tech
owner
co-owner
senior-admin
admin
junior-admin
intern
```

Конкретный API endpoint также может дополнительно проверять права вызывающего пользователя.

---

## `/add <ID/@username> <0|1> <причина>`

Добавить пользователя как скамера или сомнительного персонажа.

### Значения

```text
0 — scammer
1 — dodgy-character
```

### По Telegram ID

Скамер:

```text
/add 8237096467 0 Обманул пользователя при сделке
```

Сомнительный персонаж:

```text
/add 8237096467 1 Подозрительная активность
```

### По username

```text
/add @username 0 Обман при покупке аккаунта
```

```text
/add @username 1 Подозрительное поведение
```

Причина сохраняется в поле `ScammerReason`.

Endpoint использует один сценарий:

```text
PUT /api/admin/scammer
```

Если пользователь уже существует — его данные обновляются. Если пользователя нет — API создаёт его.

---

## `/set_role <ID/@username>`

Изменение роли пользователя.

Доступ:

```text
tech
owner
co-owner
senior-admin
```

Пример:

```text
/set_role 8237096467
```

или:

```text
/set_role @username
```

После команды бот показывает inline-клавиатуру с доступными ролями.

Выбор роли выполняется через callback `set_role:`.

Фактическое изменение отправляется в API:

```text
PUT /api/admin/user/role/:telegram
```

---

## `/complete <Telegram ID> <username>`

Дополнить существующего пользователя username'ом.

Пример:

```text
/complete 1381931882 @username
```

Первым параметром обязательно должен быть Telegram ID.

Команда используется администраторами.

Endpoint:

```text
PUT /api/admin/user/complete
```

Если username уже занят/заполнен, API возвращает соответствующую ошибку.

---

# ⚠️ Система предупреждений

## `/warn <ID/@username>`

Выдать пользователю предупреждение.

Доступ:

```text
tech
owner
co-owner
senior-admin
```

Примеры:

```text
/warn 8237096467
```

```text
/warn @username
```

Система использует лимит:

```text
0/3
1/3
2/3
3/3
```

При достижении `3/3` пользователь получает наказание, после чего варны сбрасываются.

API:

```text
PUT /api/admin/user/warn/:telegram
```

---

## `/unwarn <ID/@username>`

Снять одно предупреждение.

Пример:

```text
/unwarn 8237096467
```

или:

```text
/unwarn @username
```

API:

```text
PUT /api/admin/user/unwarn/:telegram
```

---

# 🛡️ Работа с гарантами

## `/guarantors`

Показать список всех гарантов.

```text
/guarantors
```

Отображается:

- 👤 Telegram username;
- 🆔 Telegram ID;
- 🏆 ранг;
- 🌍 регион;
- 📢 канал;
- 📊 количество пруфов;
- 👥 Trainee.

Гаранты разделяются на:

```text
🔝 Топ гаранты
🛡️ Гаранты
```

---

## `/add_g <ID/@username> <rank>`

Добавить пользователя в гаранты.

Доступные ранги:

```text
basic
top
```

### Обычный гарант

```text
/add_g 5328949040 basic
```

### Топ гарант

```text
/add_g @username top
```

Роли пользователя:

```text
basic → guarantor
top   → top-guarantor
```

---

## `/edit_g <Telegram ID>`

Редактировать существующего гаранта.

```text
/edit_g 5328949040
```

FSM позволяет изменять:

- 🏆 ранг;
- 📢 канал;
- 📊 количество пруфов;
- 🌍 регион;
- 🔄 дополнительные данные.

Для отмены:

```text
/cancel
```

---

## `/reset_g <Telegram ID>`

Сбросить дополнительные данные гаранта.

```text
/reset_g 5328949040
```

Сбрасываются:

```text
ChanelUrl
ProofsCount
Region
```

Не сбрасываются:

```text
Rank
TraineeIDs
Telegram ID
```

---

## `/del_g <ID/@username>`

Удалить гаранта.

Пример:

```text
/del_g 5328949040
```

или:

```text
/del_g @username
```

API:

```text
DELETE /api/admin/guarantors/remove/:identifier
```

---

# 👥 Работа с Trainee

Команды доступны:

```text
guarantor
top-guarantor
```

---

## `/add_trainee <Telegram ID>`

Назначить пользователя своим Trainee.

```text
/add_trainee 8237096467
```

При успешном выполнении:

1. пользователь получает роль `trainee`;
2. его Telegram ID добавляется в `TraineeIDs` текущего гаранта;
3. дубликат не добавляется.

Пользователь должен существовать в базе данных.

API:

```text
POST /api/guarantor/trainee/:telegram
```

---

## `/remove_trainee <Telegram ID>`

Удалить пользователя из своих Trainee.

```text
/remove_trainee 8237096467
```

При успешном выполнении:

1. Telegram ID удаляется из `TraineeIDs`;
2. роль пользователя возвращается на `user`.

API:

```text
DELETE /api/guarantor/trainee/:telegram
```

> В `bot.go` сейчас зарегистрированы два обработчика `/remove_trainee`. Их стоит объединить в один, чтобы избежать дублирования логики.

---

# ❌ `/cancel`

Отменить текущую FSM-операцию.

```text
/cancel
```

Используется во время многошагового редактирования гаранта и других FSM-действий.

---

# 📋 Быстрая шпаргалка

| Команда | Назначение | Доступ |
|---|---|---|
| `/start` | Регистрация | Все |
| `/check <ID/@username>` | Проверка пользователя | Все |
| `/me` | Свой профиль | Все |
| `/add <ID/@username> <0/1> <причина>` | Добавить scammer/dodgy | `canSetRole` |
| `/set_role <ID/@username>` | Изменить роль | `canSetRole` |
| `/complete <ID> <username>` | Дополнить username | Админы |
| `/warn <ID/@username>` | Выдать варн | `canSetRole` |
| `/unwarn <ID/@username>` | Снять варн | `canSetRole` |
| `/guarantors` | Список гарантов | API/общая команда |
| `/add_g <ID/@username> <rank>` | Добавить гаранта | `canSetRole` |
| `/edit_g <ID>` | Редактировать гаранта | FSM/API |
| `/reset_g <ID>` | Сбросить данные гаранта | API |
| `/del_g <ID/@username>` | Удалить гаранта | API |
| `/add_trainee <ID>` | Добавить Trainee | `guarantor`, `top-guarantor` |
| `/remove_trainee <ID>` | Удалить Trainee | `guarantor`, `top-guarantor` |
| `/cancel` | Отменить FSM | Все |

---

# 🏷️ Все роли

```text
tech
owner
co-owner
senior-admin
admin
junior-admin
intern
top-guarantor
guarantor
trainee
scammer
dodgy-character
user
```

# 🌐 API

Основные административные endpoint'ы:

```text
GET    /api/guarantors
PUT    /api/admin/guarantor/edit/:telegram
PUT    /api/admin/guarantor/reset/:telegram

POST   /api/guarantor/trainee/:telegram
DELETE /api/guarantor/trainee/:telegram
```

Также API содержит endpoints для работы с пользователями и проверки ролей.

---

# 🔐 Middleware

Доступ к административным действиям контролируется через role-based access control.

Проверка выполняется по роли пользователя:

```go
if actor.Role != database.Guarantor &&
   actor.Role != database.TopGuarantor {
    // access denied
}
```

Для административных endpoints используются соответствующие middleware.

---

# 🗃️ Модели

## User

Основные поля:

```text
ID
TgID
UserName
Role
PhotoID
UserSearched
ScammerReason
Warns
AddedScammers
CreatedAt
UpdatedAt
```

Telegram ID уникален:

```go
TgID int64 `gorm:"uniqueIndex;not null"`
```

---

## Guarantor

Основные поля:

```text
ID
TgUserID
Rank
ChanelUrl
ProofsCount
Region
TraineeIDs
CreatedAt
UpdatedAt
```

`TgUserID` используется для связи гаранта с Telegram-пользователем.

`TraineeIDs` хранит массив Telegram ID Trainee.

---

# 🧩 Repository Layer

Репозитории отвечают только за работу с базой данных.

Пример интерфейса гаранта:

```go
type GuarantorRepository interface {
    CreateGuarantor(guarantor *database.Guarantors) error
    GetAllGuarantors() ([]database.Guarantors, error)
    GetGuarantorByID(id int64) (*database.Guarantors, error)
    UpdateGuarantor(guarantor *database.Guarantors) error
    DeleteGuarantor(id int64) error
}
```

Бизнес-логика не должна находиться непосредственно в repository.

---

# ⚙️ Service Layer

Service отвечает за бизнес-логику.

Например:

```text
UserService
    ↓
UserRepository
GuarantorRepository
```

Service выполняет:

- проверки ролей;
- изменение пользователей;
- изменение гарантов;
- назначение Trainee;
- удаление Trainee;
- синхронизацию роли пользователя и гаранта.

---

# 🤖 Telegram Bot

Telegram-часть отвечает за:

- обработку команд;
- callback queries;
- FSM;
- форматирование сообщений;
- обращение к API.

Общая схема:

```text
Telegram
   ↓
Bot Handler
   ↓
API Request
   ↓
Gin API
   ↓
Service
   ↓
Repository
   ↓
PostgreSQL
```

---

# 🔄 FSM

FSM используется для многошаговых операций.

Основной сценарий:

```text
/edit_g <ID>
      ↓
Выбор поля
      ↓
Ввод значения
      ↓
Валидация
      ↓
API
      ↓
Service
      ↓
PostgreSQL
```

Поддерживаются:

- rank;
- channel;
- proofs;
- region;
- reset.

---

# 📋 Валидация

## Rank

Допустимые значения:

```text
basic
top
```

## Proofs

Количество пруфов должно быть неотрицательным числом.

## Channel

Допускается Telegram username или ссылка на Telegram-канал.

## Region

Регион ограничивается установленным приложением максимальным размером поля.

---

# 🔎 Полезные SQL-команды

## Все пользователи

```sql
SELECT *
FROM users;
```

## Пользователи по роли

```sql
SELECT tg_id, user_name, role
FROM users
WHERE role = 'scammer';
```

## Все гаранты

```sql
SELECT tg_id, user_name, role
FROM users
WHERE role IN ('guarantor', 'top-guarantor');
```

## Количество пользователей по ролям

```sql
SELECT role, COUNT(*)
FROM users
GROUP BY role
ORDER BY role;
```

## Изменение роли

```sql
UPDATE users
SET role = 'tech'
WHERE tg_id = 1646872957;
```

## Проверка пользователя

```sql
SELECT tg_id, user_name, role
FROM users
WHERE tg_id = 1646872957;
```

---

# 📥 Импорт пользователей

Для массового импорта используется:

```sql
INSERT INTO users (tg_id, user_name, role)
VALUES
    (...),
    (...)
ON CONFLICT (tg_id) DO UPDATE
SET
    user_name = EXCLUDED.user_name,
    role = EXCLUDED.role;
```

Важно использовать только роли, разрешённые `CHECK CONSTRAINT` таблицы `users`.

Корректное значение:

```text
dodgy-character
```

Некорректное:

```text
dodgy
```

---

# 🧪 Проверка после запуска

После запуска приложения рекомендуется проверить:

### 1. Контейнеры

```bash
docker compose ps
```

### 2. Логи

```bash
docker compose logs -f app
```

### 3. PostgreSQL

```bash
docker exec -it a-scam-postgres psql -U ascam -d ascam
```

### 4. Таблицы

```sql
\dt
```

### 5. Количество пользователей

```sql
SELECT COUNT(*)
FROM users;
```

### 6. Гаранты

```sql
SELECT tg_id, user_name, role
FROM users
WHERE role IN ('guarantor', 'top-guarantor');
```

---

# 🔧 Разработка

Получить зависимости:

```bash
go mod download
```

Проверить зависимости:

```bash
go mod tidy
```

Форматирование:

```bash
gofmt -w .
```

Проверка проекта:

```bash
go vet ./...
```

Сборка:

```bash
go build ./...
```

Запуск:

```bash
go run .
```

---

# 🐳 Docker Build

Собрать образ:

```bash
docker build -t a-scam-bot .
```

Запустить:

```bash
docker run --rm \
    --env-file .env \
    -p 8080:8080 \
    a-scam-bot
```

---

# 📦 GitHub Container Registry

Docker image публикуется через GitHub Actions в:

```text
ghcr.io/artem-cherevko/a-scam-bot-v2
```

Типичный workflow:

```text
git push
   ↓
GitHub Actions
   ↓
Docker Build
   ↓
GHCR
```

Для production можно использовать:

```bash
docker pull ghcr.io/artem-cherevko/a-scam-bot-v2:latest
```

---

# 🔄 CI/CD

GitHub Actions выполняет:

1. Checkout репозитория.
2. Настройку Docker Buildx.
3. Авторизацию в GHCR.
4. Генерацию Docker metadata.
5. Сборку Docker image.
6. Push image в GHCR.

Основной workflow находится:

```text
.github/workflows/docker.yml
```

---

# 🔒 Безопасность

Не коммитьте:

```text
.env
*.key
*.pem
secrets
tokens
passwords
```

Telegram Bot Token никогда не должен находиться в Git.

Для production используйте:

- environment variables;
- Docker secrets;
- GitHub Actions secrets;
- отдельные credentials для PostgreSQL.

---

# 📁 Структура проекта

Примерная структура:

```text
a-scam-bot-v2/
│
├── .github/
│   └── workflows/
│       └── docker.yml
│
├── internal/
│   ├── api/
│   ├── bot/
│   ├── config/
│   ├── database/
│   ├── repository/
│   └── service/
│
├── Dockerfile
├── docker-compose.yml
├── .dockerignore
├── .env
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

# 🧠 Принципы проекта

Основные принципы:

- разделение ответственности;
- repository/service architecture;
- role-based access control;
- валидация входных данных;
- PostgreSQL как основное хранилище;
- API как слой взаимодействия между ботом и бизнес-логикой;
- минимизация дублирования бизнес-логики;
- Docker-first deployment.

---

# 🚀 Production

Рекомендуемый production flow:

```text
Developer
    │
    ▼
Git Push
    │
    ▼
GitHub
    │
    ▼
GitHub Actions
    │
    ▼
GHCR
    │
    ▼
Docker Server
    │
    ├── A-Scam Bot
    │
    └── PostgreSQL
```

Production `.env` хранится отдельно от Git-репозитория.

---

# 📌 Команды администраторов — краткая шпаргалка

```text
/guarantors
```

Список гарантов.

```text
/add_g <ID/@username> <basic/top>
```

Добавить гаранта.

```text
/edit_g <ID>
```

Редактировать гаранта.

```text
/reset_g <ID>
```

Сбросить дополнительные данные гаранта.

```text
/add_trainee <ID>
```

Назначить Trainee.

```text
/remove_trainee <ID>
```

Удалить Trainee.

```text
/cancel
```

Отменить текущую FSM-операцию.

---

# 📄 License

Если в репозитории используется отдельная лицензия, её условия определяются файлом:

```text
LICENSE
```

Если файл отсутствует, условия использования проекта определяются владельцем репозитория.

---

# 👤 Author

**Artem Cherevko**

GitHub:

```text
github.com/artem-cherevko
```

Project:

```text
a-scam-bot-v2
```
