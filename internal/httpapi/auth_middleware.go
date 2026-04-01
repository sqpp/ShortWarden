package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/auth"
	"shortwarden/internal/store"
)

const sessionCookieName = "sw_session"
const csrfCookieName = "sw_csrf"

type authDeps struct {
	db  *pgxpool.Pool
	q   *store.Queries
	jwt *auth.JWTManager
}

func authMiddleware(d authDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1) API key auth (Authorization: ApiKey <key>)
			if raw := parseAPIKeyHeader(r.Header.Get("Authorization")); raw != "" {
				if uctx, ok := authenticateAPIKey(ctx, d, raw); ok {
					next.ServeHTTP(w, r.WithContext(uctx))
					return
				}
			}

			// 2) Cookie JWT auth
			if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
				if uid, err := d.jwt.Parse(c.Value); err == nil {
					ctx = withUserID(ctx, uid)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseAPIKeyHeader(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	const prefix = "ApiKey "
	if !strings.HasPrefix(v, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(v, prefix))
}

func authenticateAPIKey(ctx context.Context, d authDeps, rawKey string) (context.Context, bool) {
	hash := auth.HashAPIKey(rawKey)
	row, err := d.q.GetActiveAPIKeyByHash(ctx, hash)
	if err != nil {
		return ctx, false
	}
	if row.RevokedAt.Valid {
		return ctx, false
	}
	apiKeyID, ok := uuidFromPgtypeUUID(row.ID)
	if ok {
		_ = d.q.TouchAPIKeyLastUsed(ctx, row.ID)
		ctx = withAPIKeyID(ctx, apiKeyID)
	}
	userID, ok := uuidFromPgtypeUUID(row.UserID)
	if !ok {
		return ctx, false
	}
	return withUserID(ctx, userID), true
}

func requireUserID(r *http.Request) (uuid.UUID, bool) {
	return userIDFromContext(r.Context())
}

func requireCookieCSRF(w http.ResponseWriter, r *http.Request) bool {
	// If the client is using API key auth, CSRF is not applicable.
	// (Extensions/integrations typically use API keys and do not rely on cookies.)
	if parseAPIKeyHeader(r.Header.Get("Authorization")) != "" {
		return true
	}

	// If this request is authenticated via API key, CSRF is not applicable.
	// CSRF protection is only needed for cookie-authenticated browser sessions.
	if _, ok := apiKeyIDFromContext(r.Context()); ok {
		return true
	}

	// Only enforce CSRF for cookie-authenticated browser requests.
	if _, err := r.Cookie(sessionCookieName); err != nil {
		return true
	}
	// Safe methods don't require CSRF.
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	c, err := r.Cookie(csrfCookieName)
	if err != nil || c.Value == "" {
		writeError(w, http.StatusBadRequest, "missing CSRF cookie")
		return false
	}
	h := r.Header.Get("X-CSRF-Token")
	if h == "" || h != c.Value {
		writeError(w, http.StatusBadRequest, "invalid CSRF token")
		return false
	}
	return true
}

func uuidFromPgtypeUUID(v pgtype.UUID) (uuid.UUID, bool) {
	if !v.Valid {
		return uuid.UUID{}, false
	}
	id, err := uuid.FromBytes(v.Bytes[:])
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}

// Ensure we don't accidentally import unused packages.
var _ = genapi.CookieAuthScopes
var _ = pgx.ErrNoRows

