package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/auth"
	"shortwarden/internal/config"
	"shortwarden/internal/store"
)

type Handler struct {
	genapi.Unimplemented

	cfg config.Config
	db  *pgxpool.Pool
	q   *store.Queries
	jwt *auth.JWTManager
}

func NewHandler(cfg config.Config, db *pgxpool.Pool) *Handler {
	return &Handler{
		cfg: cfg,
		db:  db,
		q:   store.New(db),
		jwt: auth.NewJWTManager(cfg.JWTSecret, 7*24*time.Hour),
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req genapi.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password too short")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}

	u, err := h.q.CreateUser(r.Context(), store.CreateUserParams{
		Email:        string(req.Email),
		PasswordHash: hash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusCreated, userToAPI(u))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req genapi.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := h.q.GetUserByEmail(r.Context(), string(req.Email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if u.DisabledAt.Valid {
		writeError(w, http.StatusUnauthorized, "user disabled")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	uid, ok := uuidFromPgtypeUUID(u.ID)
	if !ok {
		writeError(w, http.StatusInternalServerError, "invalid user id")
		return
	}

	token, exp, err := h.jwt.Mint(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	setSessionCookie(w, h.cfg, token, exp)

	resp := genapi.LoginResponse{
		User:  userToAPI(u),
		Token: &token,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w, h.cfg)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing session")
		return
	}
	uid, err := h.jwt.Parse(c.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	token, exp, err := h.jwt.Mint(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	setSessionCookie(w, h.cfg, token, exp)

	u, err := h.q.GetUserByID(r.Context(), uuidToPgtype(uid))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	resp := genapi.LoginResponse{
		User:  userToAPI(u),
		Token: &token,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Csrf(w http.ResponseWriter, r *http.Request) {
	token, err := auth.NewRawAPIKey() // reuse secure random generator
	if err != nil {
		writeError(w, http.StatusInternalServerError, "csrf error")
		return
	}
	// Token only needs to be unpredictable; trim the prefix to reduce size.
	if len(token) > 3 && token[:3] == "sw_" {
		token = token[3:]
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, genapi.CsrfResponse{Token: token})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	u, err := h.q.GetUserByID(r.Context(), uuidToPgtype(uid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, userToAPI(u))
}

func (h *Handler) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.CreateApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	raw, err := auth.NewRawAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "key error")
		return
	}
	hash := auth.HashAPIKey(raw)
	row, err := h.q.CreateAPIKey(r.Context(), store.CreateAPIKeyParams{
		UserID:  uuidToPgtype(uid),
		KeyHash: hash,
		Name:    req.Name,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "api key name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	resp := genapi.CreateApiKeyResponse{
		ApiKey: apiKeyToAPI(row),
		RawKey: raw,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListApiKeys(w http.ResponseWriter, r *http.Request, params genapi.ListApiKeysParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	limit := 50
	offset := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}
	rows, err := h.q.ListAPIKeys(r.Context(), store.ListAPIKeysParams{
		UserID:  uuidToPgtype(uid),
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]genapi.ApiKey, 0, len(rows))
	for _, k := range rows {
		out = append(out, apiKeyToAPI(k))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) RevokeApiKey(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	row, err := h.q.RevokeAPIKey(r.Context(), store.RevokeAPIKeyParams{
		ID:     uuidToPgtype(uuid.UUID(id)),
		UserID: uuidToPgtype(uid),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, apiKeyToAPI(row))
}

func (h *Handler) ListApiKeyDomains(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// ensure key belongs to user
	_, err := h.q.GetAPIKeyByID(r.Context(), store.GetAPIKeyByIDParams{
		ID:     uuidToPgtype(uuid.UUID(id)),
		UserID: uuidToPgtype(uid),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	rows, err := h.q.ListAPIKeyDomains(r.Context(), uuidToPgtype(uuid.UUID(id)))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]string, 0, len(rows))
	for _, d := range rows {
		did, ok := uuidFromPgtypeUUID(d)
		if ok {
			out = append(out, did.String())
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) ReplaceApiKeyDomains(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// ensure key belongs to user
	_, err := h.q.GetAPIKeyByID(r.Context(), store.GetAPIKeyByIDParams{
		ID:     uuidToPgtype(uuid.UUID(id)),
		UserID: uuidToPgtype(uid),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	var domainIDs []string
	if err := json.NewDecoder(r.Body).Decode(&domainIDs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Replace by clearing and inserting new rows.
	if err := h.q.ReplaceAPIKeyDomains(r.Context(), uuidToPgtype(uuid.UUID(id))); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	for _, s := range domainIDs {
		did, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid domain id")
			return
		}
		// ensure domain belongs to user
		if _, err := h.q.GetDomainByID(r.Context(), store.GetDomainByIDParams{
			ID:     uuidToPgtype(did),
			UserID: uuidToPgtype(uid),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "domain not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		if err := h.q.AddAPIKeyDomain(r.Context(), store.AddAPIKeyDomainParams{
			ApiKeyID: uuidToPgtype(uuid.UUID(id)),
			DomainID: uuidToPgtype(did),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func userToAPI(u store.User) genapi.User {
	id, _ := uuidFromPgtypeUUID(u.ID)
	var disabled *time.Time
	if u.DisabledAt.Valid {
		disabled = &u.DisabledAt.Time
	}
	return genapi.User{
		Id:         openapi_types.UUID(id),
		Email:      openapi_types.Email(u.Email),
		CreatedAt:  u.CreatedAt.Time,
		DisabledAt: disabled,
	}
}

func apiKeyToAPI(k store.ApiKey) genapi.ApiKey {
	id, _ := uuidFromPgtypeUUID(k.ID)
	var lastUsed *time.Time
	if k.LastUsedAt.Valid {
		lastUsed = &k.LastUsedAt.Time
	}
	var revoked *time.Time
	if k.RevokedAt.Valid {
		revoked = &k.RevokedAt.Time
	}
	return genapi.ApiKey{
		Id:         openapi_types.UUID(id),
		Name:       k.Name,
		CreatedAt:  k.CreatedAt.Time,
		LastUsedAt: lastUsed,
		RevokedAt:  revoked,
	}
}

func setSessionCookie(w http.ResponseWriter, cfg config.Config, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		Domain:   cfg.CookieDomain,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		Domain:   cfg.CookieDomain,
		SameSite: http.SameSiteLaxMode,
	})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

