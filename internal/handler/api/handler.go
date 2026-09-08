package api

import (
	"encoding/json"
	"net/http"

	"github.com/avito-kitchen/avito_kitchen/internal/models"
	"github.com/avito-kitchen/avito_kitchen/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	store *storage.Storage
}

func New(store *storage.Storage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Client API
		r.Get("/restaurants", h.listRestaurants)
		r.Get("/restaurants/{id}/menu", h.getMenu)
		r.Get("/menu-items/{id}", h.getMenuItem)

		r.Group(func(r chi.Router) {
			r.Use(userID)
			r.Get("/cart", h.getCart)
			r.Post("/cart/items", h.addToCart)
			r.Delete("/cart/items/{id}", h.removeFromCart)
			r.Post("/orders", h.createOrder)
			r.Get("/orders/{id}", h.getOrder)
		})

		// Restaurant API
		r.Group(func(r chi.Router) {
			r.Use(h.restaurantToken)
			r.Get("/restaurant/orders", h.restaurantOrders)
			r.Patch("/restaurant/orders/{id}/status", h.updateStatus)
		})
	})
	return r
}

func (h *Handler) listRestaurants(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListRestaurants(r.Context(), r.URL.Query().Get("cuisine"))
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonOK(w, map[string]any{"items": list})
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := h.store.GetMenu(r.Context(), id)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonOK(w, map[string]any{"items": items})
}

func (h *Handler) getMenuItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.store.GetMenuItem(r.Context(), id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	jsonOK(w, models.MenuItemView{
		MenuItem:         item,
		EstimatedMinutes: storage.EstimateMinutes(item.PrepTimeMin),
	})
}

func (h *Handler) getCart(w http.ResponseWriter, r *http.Request) {
	cart, err := h.store.GetCart(r.Context(), userFrom(r))
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonOK(w, cart)
}

func (h *Handler) addToCart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MenuItemID uuid.UUID `json:"menu_item_id"`
		Quantity   int       `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.store.AddToCart(r.Context(), userFrom(r), req.MenuItemID, req.Quantity); err != nil {
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	h.getCart(w, r)
}

func (h *Handler) removeFromCart(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.RemoveFromCart(r.Context(), userFrom(r), id); err != nil {
		if err == pgx.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "not found")
			return
		}
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	h.getCart(w, r)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeliveryAddress string `json:"delivery_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DeliveryAddress == "" {
		jsonErr(w, http.StatusBadRequest, "delivery_address required")
		return
	}
	order, err := h.store.CreateOrder(r.Context(), userFrom(r), req.DeliveryAddress)
	if err != nil {
		if err.Error() == "cart is empty" {
			jsonErr(w, http.StatusBadRequest, err.Error())
			return
		}
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, order)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	order, err := h.store.GetOrder(r.Context(), id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	if order.UserID != userFrom(r) {
		jsonErr(w, http.StatusNotFound, "not found")
		return
	}
	jsonOK(w, order)
}

func (h *Handler) restaurantOrders(w http.ResponseWriter, r *http.Request) {
	rest := restaurantFrom(r)
	list, err := h.store.ListRestaurantOrders(r.Context(), rest.ID, r.URL.Query().Get("status"))
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonOK(w, map[string]any{"items": list})
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		jsonErr(w, http.StatusBadRequest, "status required")
		return
	}
	rest := restaurantFrom(r)
	if err := h.store.UpdateOrderStatus(r.Context(), rest.ID, id, req.Status); err != nil {
		if err == pgx.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "not found")
			return
		}
		jsonErr(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonOK(w, map[string]string{"status": req.Status})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
