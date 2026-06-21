# Checking Payment Status via API

Учебная задача про воркер, который проверяет статусы платежей во внешнем vendor API и обновляет состояние платежей в базе.

Изначально был дан один неоптимизированный файл с кодом на Go. Задача состояла в том, чтобы провести code review, найти проблемы в реализации и переработать код в более надежный вариант.

## Исходная логика

В системе есть платежи со статусом `pending_vendor`. Для каждого такого платежа нужно:

1. Взять платеж из таблицы `payments`.
2. Сходить во внешний vendor API по `vendor_tx_id`.
3. Получить статус платежа: `PENDING`, `PAID` или `FAILED`.
4. Обновить статус платежа в базе.
5. Если платеж оплачен, отправить сообщение во внутреннюю очередь fulfillment.

Упрощенный исходный код содержал:

- `VendorClient` для похода во внешний API.
- `FulfillmentQueue` для постановки сообщения в очередь.
- `Payment` и `PaymentPaidMessage`.
- `VerificationWorker.RunPollOnce`, который выбирал все `pending_vendor` платежи и обрабатывал их по одному.

## Основные проблемы исходного кода

В исходной версии были как синтаксические, так и архитектурные проблемы:


- Запрос выбирал все платежи со статусом `pending_vendor` без лимита, что опасно для базы и памяти.
- Не было защиты от параллельной обработки одного и того же платежа несколькими репликами приложения.
- Обработка шла последовательно, поэтому внешний API опрашивался медленно.
- Статус `PAID` сначала записывался в БД, а затем сообщение отправлялось в очередь. При ошибке очереди можно получить неконсистентное состояние.
- Не было явной стратегии повторов для зависших или неудачно обработанных платежей.
- SQL и бизнес-логика были смешаны в одном типе.
- Статусы платежей были строками без отдельного domain-типа.

## Направление исправлений

После ревью код был разнесен по слоям:

- `internal/domain` - доменные структуры и статусы платежей.
- `internal/usecase` - сценарии проверки статусов.
- `internal/postgres` - работа с PostgreSQL.
- `migrations` - SQL-миграции для таблицы `payments`.

Целевая идея обработки:

1. В транзакции выбрать ограниченную пачку платежей.
2. Заблокировать строки через `FOR UPDATE SKIP LOCKED`, чтобы несколько реплик не взяли одни и те же платежи.
3. Пометить выбранные платежи как временно заблокированные.
4. Параллельно проверить статусы во внешнем API с ограничением конкуренции через semaphore.
5. Обновить платежи пачками по итоговым статусам.
6. Периодически возвращать в обработку платежи, которые зависли в locked-состоянии слишком долго.

## Почему нужен `FOR UPDATE SKIP LOCKED`

Если приложение запущено в нескольких репликах, каждая реплика может одновременно запускать проверку платежей. Без блокировок несколько реплик могут выбрать одну и ту же строку из `payments` и несколько раз сходить во внешний API.

`FOR UPDATE SKIP LOCKED` решает эту проблему:

```sql
SELECT vendor_tx_id
FROM payments
WHERE status = 'pending_vendor'
ORDER BY updated_at
LIMIT 10
FOR UPDATE SKIP LOCKED;
```

Одна транзакция блокирует выбранные строки, а другие транзакции пропускают уже заблокированные строки и берут следующую доступную пачку.

## Миграция

Таблица `payments` хранит платежи и технические поля для обработки:

```sql
CREATE TABLE payments (
    id            BIGSERIAL PRIMARY KEY,
    order_id      TEXT        NOT NULL,
    vendor_tx_id  TEXT        NOT NULL,
    amount_cents  BIGINT      NOT NULL,
    status        TEXT        NOT NULL,
    paid_at       TIMESTAMPTZ NULL,
    is_locked     BOOLEAN     NOT NULL DEFAULT FALSE,
    time_locked   TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ NULL
);
```

Индексы нужны для быстрого поиска платежей в обработку:

```sql
CREATE INDEX idx_payments_status_updated_at ON payments (status, updated_at);
CREATE INDEX idx_payments_vendor_tx_id ON payments (vendor_tx_id);
```

## Goose

Создать новую SQL-миграцию:

```bash
goose -dir migrations create create_payments_table sql
```

Применить миграции:

```bash
goose -dir migrations postgres "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up
```

Посмотреть статус:

```bash
goose -dir migrations postgres "postgres://user:pass@localhost:5432/dbname?sslmode=disable" status
```

Откатить последнюю миграцию:

```bash
goose -dir migrations postgres "postgres://user:pass@localhost:5432/dbname?sslmode=disable" down
```

## Cron и несколько реплик

Если cron запускается внутри приложения, то при деплойменте на 5 реплик cron будет работать в каждой реплике. То есть одна и та же задача может стартовать 5 раз.

Для production-сценария лучше использовать один из вариантов:

- Kubernetes `CronJob`, который запускает отдельный job по расписанию.
- Внешний scheduler, который дергает HTTP endpoint.
- Leader election, чтобы cron выполнялся только на одной реплике.
- Distributed lock в БД или Redis.

Даже при наличии scheduler-а обработчик должен быть идемпотентным и безопасным для параллельного запуска. Для этой задачи это как раз достигается через транзакции, `FOR UPDATE SKIP LOCKED` и корректное обновление статусов.

## Что еще стоит улучшить

- Корректно обрабатывать ошибки vendor API: не всегда ошибка означает `failed`.
- Добавить retry/backoff для временных ошибок.
- Продумать outbox pattern для надежной отправки события `PaymentPaidMessage` после изменения статуса в БД.
- Добавить тесты на конкурентную обработку, зависшие lock-и и переходы статусов.
