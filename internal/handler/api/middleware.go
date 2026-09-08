package api

import (
	"context"
	"net/http"

	"github.com/avito-kitchen/avito_kitchen/internal/models"
	"github.com/google/uuid"
)

type ctxKey int

const (
	keyUser ctxKey = iota
	keyRestaurant
)

func userID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-User-ID")
		if raw == "" {
			raw = "33333333-3333-3333-3333-333333333333"
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			jsonErr(w, http.StatusBadRequest, "invalid X-User-ID")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), keyUser, id)))
	})
}

func userFrom(r *http.Request) uuid.UUID {
	return r.Context().Value(keyUser).(uuid.UUID)
}

func (h *Handler) restaurantToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Restaurant-Token")
		if token == "" {
			jsonErr(w, http.StatusUnauthorized, "token required")
			return
		}
		rest, err := h.store.GetRestaurantByToken(r.Context(), token)
		if err != nil {
			jsonErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), keyRestaurant, rest)))
	})
}

func restaurantFrom(r *http.Request) models.Restaurant {
	return r.Context().Value(keyRestaurant).(models.Restaurant)
}
