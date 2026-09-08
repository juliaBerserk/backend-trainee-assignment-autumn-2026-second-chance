package models

import (
	"time"

	"github.com/google/uuid"
)

type Restaurant struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
	Cuisine     string    `json:"cuisine"`
}

type MenuItem struct {
	ID           uuid.UUID `json:"id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        int       `json:"price"`
	PrepTimeMin  int       `json:"-"`
	IsAvailable  bool      `json:"is_available"`
}

type MenuItemView struct {
	MenuItem
	EstimatedMinutes int `json:"estimated_minutes"`
}

type CartItem struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Quantity   int       `json:"quantity"`
	Item       MenuItem  `json:"item"`
	LineTotal  int       `json:"line_total"`
}

type Cart struct {
	Items []CartItem `json:"items"`
	Total int        `json:"total"`
}

type Order struct {
	ID               uuid.UUID         `json:"id"`
	UserID           uuid.UUID         `json:"user_id"`
	DeliveryAddress  string            `json:"delivery_address"`
	TotalAmount      int               `json:"total_amount"`
	RestaurantOrders []RestaurantOrder `json:"restaurant_orders"`
	CreatedAt        time.Time         `json:"created_at"`
}

type RestaurantOrder struct {
	ID             uuid.UUID   `json:"id"`
	OrderID        uuid.UUID   `json:"order_id"`
	RestaurantID   uuid.UUID   `json:"restaurant_id"`
	RestaurantName string      `json:"restaurant_name,omitempty"`
	Status         string      `json:"status"`
	Subtotal       int         `json:"subtotal"`
	Items          []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Name       string    `json:"name"`
	Price      int       `json:"price"`
	Quantity   int       `json:"quantity"`
	LineTotal  int       `json:"line_total"`
}
