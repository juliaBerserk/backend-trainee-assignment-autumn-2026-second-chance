# Avito.Kitchen

MVP backend для заказа еды.
## Сценарии

Клиент: рестораны → меню → корзина → заказ → статус  
Ресторан: получает заказы, меняет статус (`new` → `cooking` → `delivered`)  
Фишка MVP: корзина из разных ресторанов → один заказ автоматически делится на `restaurant_orders`

## Запуск

```bash
docker compose up --build
```

API: http://localhost:8080

## Пример

```bash
# Рестораны
curl "http://localhost:8080/api/v1/restaurants?cuisine=italian"

# Меню
curl "http://localhost:8080/api/v1/restaurants/11111111-1111-1111-1111-111111111111/menu"

# Добавить пиццу и чизкейк из другого ресторана
curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Content-Type: application/json" \
  -d '{"menu_item_id":"b1111111-1111-1111-1111-111111111111","quantity":1}'

curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Content-Type: application/json" \
  -d '{"menu_item_id":"b2222222-2222-2222-2222-222222222223","quantity":1}'

# Оформить (split на 2 ресторана)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"delivery_address":"ул. Пушкина, 5"}'
```

Заголовки (MVP без auth):
- `X-User-ID` — ID пользователя (опционально, есть default)
- `X-Restaurant-Token: demo-pizza-token` — для Restaurant API

## Структура

```
cmd/api/              — HTTP-сервер
cmd/migrate/          — миграции
cmd/restaurant-demo/  — demo-интеграция ресторана
internal/models/      — структуры
internal/storage/     — SQL-запросы
internal/api/         — handlers
migrations/           — схема + seed
api/openapi.yaml
```

## Схема БД

```
restaurants ── menu_items
cart_items (user_id + menu_item_id)
orders ── restaurant_orders ── order_items
```

`prep_time_min` используется для расчёта ETA, клиенту не показывается напрямую.

## Линтер

```bash
golangci-lint run ./...
```

Конфиг: `.golangci.yml`

## OpenAPI

[api/openapi.yaml](api/openapi.yaml)