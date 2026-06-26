package seed

import (
	"context"
	"log"

	"github.com/trello-clone/backend/internal/auth"
	"github.com/trello-clone/backend/internal/config"
	"github.com/trello-clone/backend/internal/store"
)

func AdminUser(ctx context.Context, st *store.Store, tm *auth.TokenManager, cfg *config.Config) {
	if !cfg.SeedAdmin {
		return
	}
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		log.Println("seed admin: skipped (ADMIN_EMAIL and ADMIN_PASSWORD required)")
		return
	}
	hash, err := tm.HashPassword(cfg.AdminPassword)
	if err != nil {
		log.Printf("seed admin: hash password: %v", err)
		return
	}
	user, err := st.EnsureAdminUser(ctx, cfg.AdminEmail, cfg.AdminName, hash)
	if err != nil {
		log.Printf("seed admin: %v", err)
		return
	}
	log.Printf("seed admin: ready as %s (is_admin=%v)", user.Email, user.IsAdmin)
}
