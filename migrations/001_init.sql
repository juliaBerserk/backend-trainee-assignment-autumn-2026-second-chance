-- +goose Up
CREATE TYPE order_status AS ENUM ('new', 'cooking', 'delivered');

CREATE TABLE restaurants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    address     TEXT NOT NULL,
    cuisine     TEXT NOT NULL DEFAULT 'other',
    api_token   TEXT NOT NULL UNIQUE
);

CREATE TABLE menu_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id UUID NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    price         INT NOT NULL CHECK (price > 0),
    prep_time_min INT NOT NULL DEFAULT 15,
    is_available  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE cart_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL,
    menu_item_id UUID NOT NULL REFERENCES menu_items(id),
    quantity     INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    UNIQUE (user_id, menu_item_id)
);

CREATE TABLE orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    delivery_address TEXT NOT NULL,
    total_amount     INT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE restaurant_orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    restaurant_id UUID NOT NULL REFERENCES restaurants(id),
    status        order_status NOT NULL DEFAULT 'new',
    subtotal      INT NOT NULL
);

CREATE TABLE order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_order_id UUID NOT NULL REFERENCES restaurant_orders(id) ON DELETE CASCADE,
    menu_item_id        UUID NOT NULL REFERENCES menu_items(id),
    name                TEXT NOT NULL,
    price               INT NOT NULL,
    quantity            INT NOT NULL CHECK (quantity > 0)
);

-- seed
INSERT INTO restaurants (id, name, description, address, cuisine, api_token) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Pizza House', 'Итальянская пицца', 'ул. Ленина, 10', 'italian', 'demo-pizza-token'),
    ('22222222-2222-2222-2222-222222222222', 'Sushi Master', 'Японская кухня', 'пр. Мира, 25', 'japanese', 'demo-sushi-token');

INSERT INTO menu_items (id, restaurant_id, name, description, price, prep_time_min) VALUES
    ('b1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Пепперони', 'Пицца с пепперони', 59900, 20),
    ('b1111111-1111-1111-1111-111111111112', '11111111-1111-1111-1111-111111111111', 'Маргарита', 'Классическая пицца', 49900, 18),
    ('b2222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 'Филадельфия', 'Ролл с лососем', 45000, 15),
    ('b2222222-2222-2222-2222-222222222223', '22222222-2222-2222-2222-222222222222', 'Чизкейк', 'Десерт', 35000, 5);

-- +goose Down
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS restaurant_orders;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS restaurants;
DROP TYPE IF EXISTS order_status;
