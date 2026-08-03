# Core Backend — платформа визуальных новелл с ИИ-персонажами

Backend-сервис на Go для мобильно-веб платформы визуальных новелл. Отвечает за
аутентификацию, профили пользователей, сюжетную логику (истории/сцены/выборы),
кошелёк с алмазами, сессии свободного диалога с ИИ-персонажами и доставку
стриминговых ответов ИИ по WebSocket. AI Orchestrator (Python) и Flutter-клиент
— отдельные сервисы, не входящие в этот репозиторий; интеграция с ними
описана в разделе [Контракт событий NATS](#контракт-событий-nats) и
[WebSocket-протокол](#websocket-протокол).

## Стек

Go 1.23+, Fiber v2, pgx v5 + sqlc, golang-migrate, go-redis v9, nats.go
(JetStream), golang-jwt v5, argon2id, gofiber/contrib/websocket, log/slog
(JSON), caarlos0/env, testify + testcontainers-go, golangci-lint.

> **AI Orchestrator и Flutter-клиент — отдельные сервисы, не входящие в этот
> репозиторий.** Для локальной сквозной проверки WebSocket-пути без
> реального Python-сервиса в репозитории есть `cmd/mockai` — заглушка
> AI Orchestrator, см. [Проверка WebSocket-пути](#проверка-websocket-пути-с-mockai).

## Структура проекта

```
cmd/api/main.go          — точка входа, сборка зависимостей
cmd/mockai/main.go       — заглушка AI Orchestrator для локальной проверки WS (не для прода)
internal/config/         — вся конфигурация (единственное место с env-тегами)
internal/auth/           — регистрация, логин, JWT, argon2id
internal/user/           — профиль пользователя (GET/PATCH /users/me)
internal/story/          — истории, сцены, выборы, прогресс
internal/wallet/         — баланс алмазов, начисления/списания, dev-грант
internal/dialog/         — сессии свободного диалога, лимиты, шедулер
internal/realtime/       — WebSocket-хаб и подписчик на события ИИ
internal/db/sqlc/        — код, сгенерированный sqlc из internal/db/queries
migrations/              — SQL-миграции (golang-migrate, *.up.sql/*.down.sql)
pkg/httpserver/          — сборка Fiber-приложения, единый формат ошибок
pkg/middleware/          — auth/logging/ratelimit middleware
pkg/events/              — обёртка над NATS JetStream
pkg/apperr/               — единый тип ошибки {"error": {"code","message"}}
pkg/logger/               — JSON-логгер на log/slog
```

## Быстрый старт

1. Скопировать `.env.example` в `.env` и при необходимости отредактировать
   (значения по умолчанию согласованы с `docker-compose.yml`).

```bash
cp .env.example .env
```

2. Поднять зависимости (Postgres 17, Redis 7, NATS с JetStream):

```bash
make up
```

3. Накатить миграции:

```bash
make migrate
```

4. Запустить сервис:

```bash
make run
```

Сервис поднимется на `http://localhost:8080` (порт настраивается через
`SERVER_PORT`). Список всех переменных окружения — в `.env.example`.

## Проверка WebSocket-пути с mockai

Реальный AI Orchestrator — отдельный Python-сервис, не входящий в этот
репозиторий. Чтобы вручную проверить путь `NATS → realtime.Subscriber → WS`
без него, в комплекте есть заглушка `cmd/mockai`: она слушает те же
NATS-события, что публикует Core Backend (`player.choice.made`,
`dialog.session.started`, `dialog.message.sent`, `dialog.session.ended`), и
отвечает `ai.*`-событиями с фейковым текстом и искусственными задержками
(эмуляция латентности LLM и стриминга токенов). **Это только для локальной
разработки** — никогда не запускайте её рядом с настоящим AI Orchestrator на
одном NATS.

1. Поднять зависимости и накатить миграции (см. [Быстрый старт](#быстрый-старт)).
2. В одном терминале — основной сервис:

```bash
make run
```

3. В соседнем терминале — заглушка AI Orchestrator:

```bash
make run-mockai
```

Оба процесса используют один и тот же `NATS_URL` из `.env` и не конфликтуют
между собой: `internal/realtime.Subscriber` слушает подписку `ai.>` под
durable-именем `core_backend_ai_events`, а `mockai` — подписку
`player.>`/`dialog.>` под именем `mockai_consumer`, это разные consumer'ы на
одном стриме `VN_EVENTS`.

4. Зарегистрироваться и получить токен:

```bash
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"ws-test@example.com","password":"testpass123"}'
```

Скопировать `access_token` из ответа — он понадобится для WS-хэндшейка и
дальнейших запросов.

5. Открыть WebSocket-соединение (нужен [websocat](https://github.com/vi/websocat),
   либо любой другой WS-клиент, например расширение браузера или `wscat`):

```bash
websocat "ws://localhost:8080/ws?token=<ACCESS_TOKEN>"
```

Соединение остаётся открытым и печатает в консоль всё, что присылает
сервер — держите этот терминал видимым во время следующих шагов.

6. **Проверка сценарного пути** (`player.choice.made` → `ai.response.ready`):
   создать историю/сцену/выбор напрямую в БД (админ-панели нет — это вне
   рамок этого этапа) и вызвать `POST /scenes/:id/choice` с `Authorization:
   Bearer <ACCESS_TOKEN>`. В окне с `websocat` в течение 1-2 секунд должно
   появиться:

```json
{"type":"ai_response_ready","scene_id":"...","text":"..."}
```

7. **Проверка свободного диалога** (`dialog.message.sent` → стрим чанков →
   `ai.dialog.response.complete`): вызвать `POST /dialog/sessions` (сцена
   должна иметь `free_dialog_enabled = true`), затем `POST
   /dialog/sessions/:id/messages` с телом `{"text": "Привет!"}`. В окне
   `websocat` должны последовательно появиться 4-6 сообщений вида

```json
{"type":"ai_response_chunk","session_id":"...","chunk":"..."}
```

   а следом — одно:

```json
{"type":"ai_response_complete","session_id":"...","text":"..."}
```

8. **Проверка завершения сессии** (`dialog.session.ended` →
   `ai.session.finalized`): вызвать `POST /dialog/sessions/:id/end`. В
   логах `mockai` (не в WS!) появится подтверждение публикации
   `ai.session.finalized` — у этого события нет прямого клиентского типа
   сообщения в WS-протоколе, `internal/realtime.Subscriber` просто
   логирует его получение без ошибок, в окно `websocat` ничего
   дополнительно не прилетает.

Логи обоих процессов — структурированный JSON (`log/slog`): в логе `mockai`
видно каждое полученное `player.*`/`dialog.*` событие и каждое
опубликованное `ai.*` событие, что удобно для отладки, если что-то не
доходит до WS.

## Makefile-команды

| Команда | Назначение |
|---|---|
| `make up` / `make down` | поднять/остановить Postgres, Redis, NATS через docker-compose |
| `make migrate` | накатить все миграции |
| `make migrate-down` | откатить последнюю миграцию |
| `make build` | собрать бинарник в `bin/api` |
| `make run` | собрать и запустить сервис |
| `make run-mockai` | запустить заглушку AI Orchestrator (`cmd/mockai`) — нужна для проверки WS-пути, см. ниже |
| `make sqlc-generate` | перегенерировать `internal/db/sqlc` из `internal/db/queries` |
| `make test-unit` | юнит-тесты сервисного слоя (без Docker) |
| `make test-integration` | интеграционные тесты через testcontainers-go (нужен Docker) |
| `make test` | `test-unit` + `test-integration` |
| `make lint` | `golangci-lint run ./...` |

## Тестирование

- **Юнит-тесты** (`internal/*/service_test.go`) мокируют репозитории и
  внешние зависимости (кошелёк, публикатор событий, сцены) через
  `testify/mock` и не требуют Docker: `go test ./...`.
- **Интеграционные тесты** (`internal/*/integration_test.go`, файлы собраны
  под тегом `integration`) поднимают реальный Postgres в контейнере через
  testcontainers-go, накатывают миграции и гоняют сервисный слой поверх
  настоящей БД: `go test -tags integration -p 1 ./... -run Integration`.
  Флаг `-p 1` сериализует запуск пакетов — параллельный подъём нескольких
  Postgres-контейнеров на слабой машине/Docker Desktop может приводить к
  таймаутам при старте контейнера.
- Отдельный интеграционный тест (`internal/wallet/integration_test.go`,
  `TestWalletIntegration_ConcurrentDebitsNeverGoNegative`) параллельно шлёт
  списания, превышающие баланс, и проверяет, что баланс, посчитанный на лету
  агрегатом по `wallet_transactions`, никогда не становится отрицательным —
  списания сериализуются советным (advisory) блокировкой на уровне
  пользователя в рамках одной транзакции.

## Единая конфигурация

Все настраиваемые значения читаются только из `internal/config/config.go`
через `env`-теги (`caarlos0/env`); больше ни один файл проекта не вызывает
`os.Getenv`. Обязательные секреты (`POSTGRES_PASSWORD`, `JWT_SECRET`, ...)
помечены `required` — при отсутствии сервис падает с понятной ошибкой на
старте, а не работает с пустыми значениями.

## Формат ошибок

Единый на весь сервис JSON-формат ошибок:

```json
{ "error": { "code": "invalid_credentials", "message": "Неверный email или пароль" } }
```

## API

Базовый URL: `http://localhost:<SERVER_PORT>`. Авторизация — заголовок
`Authorization: Bearer <access_token>` (для `/ws` — токен также принимается
как query-параметр `?token=...`, так как браузерный WebSocket-клиент не
может выставить заголовок при хэндшейке).

| Метод | Путь | Auth | Назначение |
|---|---|---|---|
| POST | `/auth/register` | нет | регистрация: email, password |
| POST | `/auth/login` | нет | вход, возвращает access+refresh токены |
| POST | `/auth/refresh` | нет | обновление access-токена по refresh-токену |
| GET | `/users/me` | да | профиль текущего пользователя |
| PATCH | `/users/me` | да | обновление персонализации (имя, пол, хобби, тема, язык) |
| GET | `/stories` | да | список опубликованных историй |
| GET | `/stories/:id/progress` | да | текущий прогресс игрока по истории (создаётся, если нет) |
| GET | `/scenes/:id` | да | данные сцены (фон, диалог-каркас, доступные выборы) |
| POST | `/scenes/:id/choice` | да | отправить выбор игрока → сохранить прогресс + событие в NATS |
| GET | `/wallet/balance` | да | текущий баланс алмазов |
| POST | `/dev/wallet/grant` | да, только если `ENABLE_DEV_ENDPOINTS=true` | ручное начисление алмазов вместо платежей (404, если выключено) |
| POST | `/dialog/sessions` | да | начать сессию свободного диалога (`character_id`, `scene_id`) |
| POST | `/dialog/sessions/:id/messages` | да | отправить сообщение в свободном диалоге |
| POST | `/dialog/sessions/:id/end` | да | принудительно завершить сессию |
| GET | `/ws` | да (токен в query или заголовке при хэндшейке) | WebSocket для стриминговых ответов ИИ и уведомлений |

## Контракт событий NATS

Стрим JetStream `VN_EVENTS` покрывает подстримы `player.>`, `dialog.>`, `ai.>`
(создаётся идемпотентно при старте сервиса).

**Публикует Core Backend:**

- `player.choice.made` — `{ user_id, story_id, scene_id, choice_id }`
- `dialog.session.started` — `{ session_id, user_id, character_id, scene_id }`
- `dialog.message.sent` — `{ session_id, message_id, user_id, character_id, text, remaining_limit }`
- `dialog.session.ended` — `{ session_id, reason }`

**Слушает Core Backend** (публикует AI Orchestrator, не входящий в этот репозиторий):

- `ai.response.ready` — `{ user_id, scene_id, response_text }` → доставляется в WS
- `ai.dialog.response.chunk` — `{ session_id, message_id, chunk_text }` → доставляется в WS
- `ai.dialog.response.complete` — `{ session_id, message_id, full_text }` → сохраняется в `dialog_messages` (sender=character) и доставляется в WS
- `ai.session.finalized` — `{ session_id }` → лог-подтверждение

## WebSocket-протокол

Клиент подключается к `/ws?token=<jwt>`. Сервер отправляет JSON-сообщения:

```json
{ "type": "ai_response_chunk", "session_id": "...", "chunk": "..." }
{ "type": "ai_response_complete", "session_id": "...", "text": "..." }
{ "type": "session_ended", "session_id": "...", "reason": "time_limit" }
{ "type": "ai_response_ready", "scene_id": "...", "text": "..." }
```

`realtime.Hub` хранит одно соединение на пользователя (`map[userID]*conn`,
защищённый мьютексом) и метод `SendToUser(userID string, payload []byte) error`.

## Лимиты свободного диалога

`dialog.SessionLimiter.CheckLimit` сравнивает `time`-лимит с `now() -
started_at`, а `messages`-лимит — с `message_count`; при превышении вызывающий
код завершает сессию с причиной `time_limit`/`message_limit`.
`dialog.SessionScheduler` — фоновая горутина с `time.Ticker` на интервале
`DIALOG_SESSION_CHECK_INTERVAL`, которая находит истёкшие по времени активные
сессии и завершает их (страховка на случай, если клиент не шлёт больше
сообщений и не вызывает `/end`).

## Кошелёк без платежей

Баланс никогда не хранится отдельным полем — он всегда пересчитывается как
`SELECT COALESCE(SUM(amount), 0) FROM wallet_transactions WHERE user_id = $1`,
что исключает рассинхронизацию. Списание сериализуется per-user advisory-блокировкой
(`pg_advisory_xact_lock`) в рамках одной транзакции: обычная `SELECT ... FOR
UPDATE` здесь не подходит, так как в Postgres она не допускается с
агрегатными функциями, а блокировка существующих строк не блокирует
конкурентный `INSERT`. `/dev/wallet/grant` — временная замена реального
платёжного провайдера, доступна только при `ENABLE_DEV_ENDPOINTS=true`
(иначе route не регистрируется и отвечает 404).

## Что не реализовано на этом этапе

Интеграция с реальным платёжным провайдером (вместо неё — dev-эндпоинт выше)
и админ-панель — по условиям задачи.
