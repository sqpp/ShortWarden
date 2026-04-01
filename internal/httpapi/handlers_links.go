package httpapi

import (
	"crypto/rand"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/store"
)

var aliasRe = regexp.MustCompile(`^[A-Za-z0-9_-]{4,32}$`)

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !isValidURL(req.TargetUrl) {
		writeError(w, http.StatusBadRequest, "invalid target_url")
		return
	}

	alias := ""
	if req.Alias != nil {
		alias = strings.TrimSpace(*req.Alias)
		if alias == "" {
			writeError(w, http.StatusBadRequest, "alias must not be empty")
			return
		}
		if !aliasRe.MatchString(alias) {
			writeError(w, http.StatusBadRequest, "invalid alias")
			return
		}
	} else {
		var err error
		alias, err = generateAlias()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "alias generation error")
			return
		}
	}

	domainID := pgtype.UUID{Valid: false}
	if req.DomainId != nil {
		domainID = uuidToPgtype(uuid.UUID(*req.DomainId))
	}
	var expires pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expires = pgtype.Timestamptz{Time: req.ExpiresAt.UTC(), Valid: true}
	}
	var title pgtype.Text
	if req.Title != nil {
		title = pgtype.Text{String: *req.Title, Valid: true}
	}
	tags := []string{}
	if req.Tags != nil {
		tags = *req.Tags
	}

	// Apply per-domain default tags when tags omitted.
	if req.Tags == nil {
		// Determine effective domain: explicit domain_id or user's primary domain.
		effective := domainID
		if !effective.Valid {
			if d, err := h.q.GetPrimaryDomain(r.Context(), uuidToPgtype(uid)); err == nil {
				effective = d.ID
			}
		}
		if effective.Valid {
			if d, err := h.q.GetDomainByID(r.Context(), store.GetDomainByIDParams{
				ID:     effective,
				UserID: uuidToPgtype(uid),
			}); err == nil {
				tags = d.DefaultTags
			}
		}
	}

	// If request uses API key auth and the key is restricted, enforce domain allow-list.
	if apiKeyID, ok := apiKeyIDFromContext(r.Context()); ok {
		// Only enforce when allow-list is non-empty.
		cnt, err := h.q.CountAPIKeyDomains(r.Context(), uuidToPgtype(apiKeyID))
		if err == nil && cnt > 0 {
			// Determine effective domain for this link (explicit domain_id or user's primary domain).
			effective := domainID
			if !effective.Valid {
				if d, err := h.q.GetPrimaryDomain(r.Context(), uuidToPgtype(uid)); err == nil {
					effective = d.ID
				}
			}
			if !effective.Valid {
				writeError(w, http.StatusBadRequest, "domain_id required for this api key")
				return
			}
			allowed, err := h.q.ListAPIKeyDomains(r.Context(), uuidToPgtype(apiKeyID))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "db error")
				return
			}
			isAllowed := false
			for _, a := range allowed {
				if a.Valid && a.Bytes == effective.Bytes {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				writeError(w, http.StatusBadRequest, "api key not allowed for this domain")
				return
			}
		}
	}

	row, err := h.q.CreateLink(r.Context(), store.CreateLinkParams{
		UserID:    uuidToPgtype(uid),
		DomainID:  domainID,
		Alias:     alias,
		TargetUrl: req.TargetUrl,
		Title:     title,
		Tags:      tags,
		ExpiresAt: expires,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "alias already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	apiLink := h.linkToAPI(r.Context(), requestScheme(r), uid, row)
	writeJSON(w, http.StatusCreated, apiLink)
}

func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request, params genapi.ListLinksParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	limit := 50
	offset := 0
	includeDeleted := false
	tag := ""
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}
	if params.IncludeDeleted != nil {
		includeDeleted = *params.IncludeDeleted
	}
	if params.Tag != nil {
		tag = strings.TrimSpace(*params.Tag)
	}

	rows, err := h.q.ListLinks(r.Context(), store.ListLinksParams{
		UserID:         uuidToPgtype(uid),
		Limit:          int32(limit),
		Column3:        includeDeleted,
		Offset:         int32(offset),
		Column5:        tag,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]genapi.Link, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.linkToAPI(r.Context(), requestScheme(r), uid, row))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) GetLink(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	row, err := h.q.GetLinkByID(r.Context(), store.GetLinkByIDParams{
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
	writeJSON(w, http.StatusOK, h.linkToAPI(r.Context(), requestScheme(r), uid, row))
}

func (h *Handler) UpdateLink(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.UpdateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !isValidURL(req.TargetUrl) {
		writeError(w, http.StatusBadRequest, "invalid target_url")
		return
	}

	// Keep existing alias unless provided.
	existing, err := h.q.GetLinkByID(r.Context(), store.GetLinkByIDParams{
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

	alias := existing.Alias
	if req.Alias != nil {
		a := strings.TrimSpace(*req.Alias)
		if a == "" || !aliasRe.MatchString(a) {
			writeError(w, http.StatusBadRequest, "invalid alias")
			return
		}
		alias = a
	}

	domainID := existing.DomainID
	if req.DomainId != nil {
		domainID = uuidToPgtype(uuid.UUID(*req.DomainId))
	}
	var expires pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expires = pgtype.Timestamptz{Time: req.ExpiresAt.UTC(), Valid: true}
	} else {
		expires = existing.ExpiresAt
	}
	var title pgtype.Text
	if req.Title != nil {
		title = pgtype.Text{String: *req.Title, Valid: true}
	} else {
		title = existing.Title
	}
	tags := existing.Tags
	if req.Tags != nil {
		tags = *req.Tags
	}

	row, err := h.q.UpdateLink(r.Context(), store.UpdateLinkParams{
		ID:        uuidToPgtype(uuid.UUID(id)),
		UserID:    uuidToPgtype(uid),
		DomainID:  domainID,
		Alias:     alias,
		TargetUrl: req.TargetUrl,
		Title:     title,
		Tags:      tags,
		ExpiresAt: expires,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "alias already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, h.linkToAPI(r.Context(), requestScheme(r), uid, row))
}

func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	row, err := h.q.SoftDeleteLink(r.Context(), store.SoftDeleteLinkParams{
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
	writeJSON(w, http.StatusOK, h.linkToAPI(r.Context(), requestScheme(r), uid, row))
}

func (h *Handler) linkToAPI(ctx context.Context, scheme string, uid uuid.UUID, l store.Link) genapi.Link {
	id, _ := uuidFromPgtypeUUID(l.ID)

	var domainID *openapi_types.UUID
	if l.DomainID.Valid {
		did, _ := uuidFromPgtypeUUID(l.DomainID)
		tmp := openapi_types.UUID(did)
		domainID = &tmp
	}
	var expires *time.Time
	if l.ExpiresAt.Valid {
		expires = &l.ExpiresAt.Time
	}
	var deleted *time.Time
	if l.DeletedAt.Valid {
		deleted = &l.DeletedAt.Time
	}
	var title *string
	if l.Title.Valid {
		title = &l.Title.String
	}
	tags := l.Tags
	base := h.resolveShortBaseForLink(ctx, scheme, uid, l)
	short := strings.TrimRight(base, "/") + "/r/" + l.Alias

	return genapi.Link{
		Id:        openapi_types.UUID(id),
		DomainId:  domainID,
		Alias:     l.Alias,
		TargetUrl: l.TargetUrl,
		Title:     title,
		Tags:      &tags,
		CreatedAt: l.CreatedAt.Time,
		UpdatedAt: l.UpdatedAt.Time,
		ExpiresAt: expires,
		DeletedAt: deleted,
		ShortUrl:  &short,
	}
}

func isValidURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	// Prevent obvious whitespace/control characters.
	return !strings.ContainsAny(raw, " \t\r\n")
}

func generateAlias() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const n = 8
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := 0; i < n; i++ {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}

