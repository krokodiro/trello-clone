package middleware

import (
	"net/http"

	"github.com/trello-clone/backend/internal/store"
)

func RequireVerified(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			user, err := s.GetUserByID(r.Context(), userID)
			if err != nil || user == nil || user.EmailVerifiedAt == nil {
				http.Error(w, `{"error":"email not verified"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
