package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/auth"
	"shortwarden/internal/store"
)

const dnsChallengePrefix = "_shortwarden-challenge."

func (h *Handler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	hostname, err := normalizeHostname(req.Hostname)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid hostname")
		return
	}
	raw, err := auth.NewRawAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	token := strings.TrimPrefix(raw, "sw_")

	row, err := h.q.CreateDomain(r.Context(), store.CreateDomainParams{
		UserID:   uuidToPgtype(uid),
		Hostname: hostname,
		DnsToken: token,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "domain already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	txtName := dnsChallengePrefix + hostname
	resp := genapi.CreateDomainResponse{
		Domain:    domainToAPI(row),
		TxtName:   txtName,
		TxtValue:  token,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListDomains(w http.ResponseWriter, r *http.Request, params genapi.ListDomainsParams) {
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
	rows, err := h.q.ListDomains(r.Context(), store.ListDomainsParams{
		UserID:  uuidToPgtype(uid),
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]genapi.Domain, 0, len(rows))
	for _, d := range rows {
		out = append(out, domainToAPI(d))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) VerifyDomain(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	d, err := h.q.GetDomainByID(r.Context(), store.GetDomainByIDParams{
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

	txtName := dnsChallengePrefix + d.Hostname
	txts, err := net.LookupTXT(txtName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "dns lookup failed")
		return
	}
	found := false
	for _, v := range txts {
		if strings.TrimSpace(v) == d.DnsToken {
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusBadRequest, "dns token not found")
		return
	}

	updated, err := h.q.MarkDomainVerified(r.Context(), store.MarkDomainVerifiedParams{
		ID:     uuidToPgtype(uuid.UUID(id)),
		UserID: uuidToPgtype(uid),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, domainToAPI(updated))
}

func (h *Handler) SetPrimaryDomain(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// Must be verified to become primary.
	_, err := h.q.GetVerifiedDomainByID(r.Context(), store.GetVerifiedDomainByIDParams{
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
	if err := h.q.SetPrimaryDomain(r.Context(), store.SetPrimaryDomainParams{
		UserID: uuidToPgtype(uid),
		ID:     uuidToPgtype(uuid.UUID(id)),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteDomain(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	d, err := h.q.GetDomainByID(r.Context(), store.GetDomainByIDParams{
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
	if d.IsPrimary {
		writeError(w, http.StatusBadRequest, "cannot delete primary domain")
		return
	}
	if err := h.q.DeleteDomain(r.Context(), store.DeleteDomainParams{
		ID:     uuidToPgtype(uuid.UUID(id)),
		UserID: uuidToPgtype(uid),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ReplaceDomainDefaultTags(w http.ResponseWriter, r *http.Request, id genapi.IdParam) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// Ensure domain exists and belongs to user
	_, err := h.q.GetDomainByID(r.Context(), store.GetDomainByIDParams{
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

	var tags []string
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	updated, err := h.q.UpdateDomainDefaultTags(r.Context(), store.UpdateDomainDefaultTagsParams{
		ID:         uuidToPgtype(uuid.UUID(id)),
		UserID:     uuidToPgtype(uid),
		DefaultTags: tags,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, domainToAPI(updated))
}

func domainToAPI(d store.Domain) genapi.Domain {
	id, _ := uuidFromPgtypeUUID(d.ID)
	var verifiedAt *time.Time
	if d.VerifiedAt.Valid {
		verifiedAt = &d.VerifiedAt.Time
	}
	status := genapi.DomainStatus(d.Status)
	if !status.Valid() {
		status = genapi.Pending
	}
	return genapi.Domain{
		Id:         openapi_types.UUID(id),
		Hostname:   d.Hostname,
		IsPrimary:  d.IsPrimary,
		Status:     status,
		DnsToken:   d.DnsToken,
		CreatedAt:  d.CreatedAt.Time,
		VerifiedAt: verifiedAt,
	}
}

func normalizeHostname(in string) (string, error) {
	s := strings.TrimSpace(in)
	if s == "" {
		return "", errors.New("empty")
	}
	// If user pastes a URL, parse and extract host.
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", err
		}
		s = u.Hostname()
	}
	// Remove any path fragments accidentally included.
	if strings.ContainsAny(s, "/?#") {
		u, err := url.Parse("https://" + s)
		if err == nil && u.Hostname() != "" {
			s = u.Hostname()
		}
	}
	s = strings.TrimSuffix(s, ".")
	s = strings.ToLower(s)
	// Basic sanity.
	if len(s) < 1 || len(s) > 253 {
		return "", errors.New("bad length")
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return "", errors.New("whitespace")
	}
	return s, nil
}

func (h *Handler) resolveShortBaseForLink(rctx context.Context, scheme string, uid uuid.UUID, link store.Link) string {
	// Prefer link-specific verified domain.
	if link.DomainID.Valid {
		if d, err := h.q.GetVerifiedDomainByID(rctx, store.GetVerifiedDomainByIDParams{
			ID:     link.DomainID,
			UserID: uuidToPgtype(uid),
		}); err == nil {
			return scheme + "://" + d.Hostname
		}
	}
	// Else prefer user's primary verified domain.
	if d, err := h.q.GetPrimaryDomain(rctx, uuidToPgtype(uid)); err == nil {
		if d.Status == "verified" {
			return scheme + "://" + d.Hostname
		}
	}
	return h.cfg.PublicBaseURL
}

