package storage

import (
	"context"
	"fmt"

	"github.com/avito-kitchen/avito_kitchen/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Storage {
	return &Storage{db: db}
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Storage) ListRestaurants(ctx context.Context, cuisine string) ([]models.Restaurant, error) {
	query := `SELECT id, name, description, address, cuisine FROM restaurants`
	args := []any{}
	if cuisine != "" {
		query += ` WHERE cuisine = $1`
		args = append(args, cuisine)
	}
	query += ` ORDER BY name`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Restaurant
	for rows.Next() {
		var r models.Restaurant
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Address, &r.Cuisine); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *Storage) GetMenu(ctx context.Context, restaurantID uuid.UUID) ([]models.MenuItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, restaurant_id, name, description, price, prep_time_min, is_available
		FROM menu_items WHERE restaurant_id = $1 AND is_available = TRUE ORDER BY name
	`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	for rows.Next() {
		var m models.MenuItem
		if err := rows.Scan(&m.ID, &m.RestaurantID, &m.Name, &m.Description, &m.Price, &m.PrepTimeMin, &m.IsAvailable); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (s *Storage) GetMenuItem(ctx context.Context, id uuid.UUID) (models.MenuItem, error) {
	var m models.MenuItem
	err := s.db.QueryRow(ctx, `
		SELECT id, restaurant_id, name, description, price, prep_time_min, is_available
		FROM menu_items WHERE id = $1
	`, id).Scan(&m.ID, &m.RestaurantID, &m.Name, &m.Description, &m.Price, &m.PrepTimeMin, &m.IsAvailable)
	return m, err
}

func (s *Storage) GetCart(ctx context.Context, userID uuid.UUID) (models.Cart, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ci.id, ci.menu_item_id, ci.quantity,
		       mi.id, mi.restaurant_id, mi.name, mi.description, mi.price, mi.prep_time_min, mi.is_available
		FROM cart_items ci
		JOIN menu_items mi ON mi.id = ci.menu_item_id
		WHERE ci.user_id = $1
	`, userID)
	if err != nil {
		return models.Cart{}, err
	}
	defer rows.Close()

	cart := models.Cart{}
	for rows.Next() {
		var ci models.CartItem
		if err := rows.Scan(
			&ci.ID, &ci.MenuItemID, &ci.Quantity,
			&ci.Item.ID, &ci.Item.RestaurantID, &ci.Item.Name, &ci.Item.Description,
			&ci.Item.Price, &ci.Item.PrepTimeMin, &ci.Item.IsAvailable,
		); err != nil {
			return cart, err
		}
		ci.LineTotal = ci.Item.Price * ci.Quantity
		cart.Total += ci.LineTotal
		cart.Items = append(cart.Items, ci)
	}
	return cart, rows.Err()
}

func (s *Storage) AddToCart(ctx context.Context, userID, menuItemID uuid.UUID, qty int) error {
	if qty <= 0 {
		qty = 1
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO cart_items (user_id, menu_item_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, menu_item_id) DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
	`, userID, menuItemID, qty)
	return err
}

func (s *Storage) RemoveFromCart(ctx context.Context, userID, cartItemID uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM cart_items WHERE id = $1 AND user_id = $2`, cartItemID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type restaurantGroup struct {
	restaurantID uuid.UUID
	items        []models.CartItem
	subtotal     int
	maxPrep      int
}

func (s *Storage) CreateOrder(ctx context.Context, userID uuid.UUID, address string) (models.Order, error) {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return models.Order{}, err
	}
	if len(cart.Items) == 0 {
		return models.Order{}, fmt.Errorf("cart is empty")
	}

	groups := groupByRestaurant(cart.Items)
	var order models.Order

	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO orders (user_id, delivery_address, total_amount) VALUES ($1, $2, $3)
			RETURNING id, created_at
		`, userID, address, cart.Total).Scan(&order.ID, &order.CreatedAt); err != nil {
			return err
		}
		order.UserID = userID
		order.DeliveryAddress = address
		order.TotalAmount = cart.Total

		for _, g := range groups {
			var ro models.RestaurantOrder
			if err := tx.QueryRow(ctx, `
				INSERT INTO restaurant_orders (order_id, restaurant_id, status, subtotal)
				VALUES ($1, $2, 'new', $3)
				RETURNING id
			`, order.ID, g.restaurantID, g.subtotal).Scan(&ro.ID); err != nil {
				return err
			}
			ro.OrderID = order.ID
			ro.RestaurantID = g.restaurantID
			ro.Status = "new"
			ro.Subtotal = g.subtotal

			for _, ci := range g.items {
				var oi models.OrderItem
				lineTotal := ci.Item.Price * ci.Quantity
				if err := tx.QueryRow(ctx, `
					INSERT INTO order_items (restaurant_order_id, menu_item_id, name, price, quantity)
					VALUES ($1, $2, $3, $4, $5) RETURNING id
				`, ro.ID, ci.MenuItemID, ci.Item.Name, ci.Item.Price, ci.Quantity).Scan(&oi.ID); err != nil {
					return err
				}
				oi.MenuItemID = ci.MenuItemID
				oi.Name = ci.Item.Name
				oi.Price = ci.Item.Price
				oi.Quantity = ci.Quantity
				oi.LineTotal = lineTotal
				ro.Items = append(ro.Items, oi)
			}
			order.RestaurantOrders = append(order.RestaurantOrders, ro)
		}

		_, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE user_id = $1`, userID)
		return err
	})
	return order, err
}

func groupByRestaurant(items []models.CartItem) []restaurantGroup {
	m := map[uuid.UUID]*restaurantGroup{}
	order := []uuid.UUID{}
	for _, ci := range items {
		g, ok := m[ci.Item.RestaurantID]
		if !ok {
			g = &restaurantGroup{restaurantID: ci.Item.RestaurantID}
			m[ci.Item.RestaurantID] = g
			order = append(order, ci.Item.RestaurantID)
		}
		g.items = append(g.items, ci)
		g.subtotal += ci.LineTotal
		if ci.Item.PrepTimeMin > g.maxPrep {
			g.maxPrep = ci.Item.PrepTimeMin
		}
	}
	result := make([]restaurantGroup, 0, len(order))
	for _, id := range order {
		result = append(result, *m[id])
	}
	return result
}

func (s *Storage) GetOrder(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	var order models.Order
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, delivery_address, total_amount, created_at FROM orders WHERE id = $1
	`, orderID).Scan(&order.ID, &order.UserID, &order.DeliveryAddress, &order.TotalAmount, &order.CreatedAt)
	if err != nil {
		return order, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT ro.id, ro.order_id, ro.restaurant_id, r.name, ro.status, ro.subtotal
		FROM restaurant_orders ro JOIN restaurants r ON r.id = ro.restaurant_id
		WHERE ro.order_id = $1
	`, orderID)
	if err != nil {
		return order, err
	}
	defer rows.Close()

	for rows.Next() {
		var ro models.RestaurantOrder
		if err := rows.Scan(&ro.ID, &ro.OrderID, &ro.RestaurantID, &ro.RestaurantName, &ro.Status, &ro.Subtotal); err != nil {
			return order, err
		}
		itemRows, err := s.db.Query(ctx, `
			SELECT id, menu_item_id, name, price, quantity FROM order_items WHERE restaurant_order_id = $1
		`, ro.ID)
		if err != nil {
			return order, err
		}
		for itemRows.Next() {
			var oi models.OrderItem
			if err := itemRows.Scan(&oi.ID, &oi.MenuItemID, &oi.Name, &oi.Price, &oi.Quantity); err != nil {
				itemRows.Close()
				return order, err
			}
			oi.LineTotal = oi.Price * oi.Quantity
			ro.Items = append(ro.Items, oi)
		}
		itemRows.Close()
		order.RestaurantOrders = append(order.RestaurantOrders, ro)
	}
	return order, rows.Err()
}

func (s *Storage) GetRestaurantByToken(ctx context.Context, token string) (models.Restaurant, error) {
	var r models.Restaurant
	err := s.db.QueryRow(ctx, `
		SELECT id, name, description, address, cuisine FROM restaurants WHERE api_token = $1
	`, token).Scan(&r.ID, &r.Name, &r.Description, &r.Address, &r.Cuisine)
	return r, err
}

func (s *Storage) ListRestaurantOrders(ctx context.Context, restaurantID uuid.UUID, status string) ([]models.RestaurantOrder, error) {
	query := `
		SELECT ro.id, ro.order_id, ro.restaurant_id, r.name, ro.status, ro.subtotal
		FROM restaurant_orders ro JOIN restaurants r ON r.id = ro.restaurant_id
		WHERE ro.restaurant_id = $1
	`
	args := []any{restaurantID}
	if status != "" {
		query += ` AND ro.status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY ro.id DESC`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.RestaurantOrder
	for rows.Next() {
		var ro models.RestaurantOrder
		if err := rows.Scan(&ro.ID, &ro.OrderID, &ro.RestaurantID, &ro.RestaurantName, &ro.Status, &ro.Subtotal); err != nil {
			return nil, err
		}
		list = append(list, ro)
	}
	return list, rows.Err()
}

func (s *Storage) UpdateOrderStatus(ctx context.Context, restaurantID, restaurantOrderID uuid.UUID, status string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE restaurant_orders SET status = $1 WHERE id = $2 AND restaurant_id = $3
	`, status, restaurantOrderID, restaurantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func EstimateMinutes(prepTime int) int {
	return prepTime + 25 // prep + доставка (упрощённо)
}
