package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/auth"
	"shortwarden/internal/store"
)

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request, params genapi.ListTagsParams) {
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

	// curated tags
	curatedRows, err := h.q.ListUserTags(r.Context(), store.ListUserTagsParams{
		UserID: uuidToPgtype(uid),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	curatedSet := map[string]bool{}
	for _, t := range curatedRows {
		curatedSet[t.Name] = true
	}

	// counts from links
	countRows, err := h.q.TagCountsFromLinks(r.Context(), store.TagCountsFromLinksParams{
		UserID: uuidToPgtype(uid),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	out := make([]genapi.Tag, 0, len(countRows)+len(curatedRows))
	seen := map[string]bool{}
	for _, row := range countRows {
		name := row.Name
		curated := curatedSet[name]
		out = append(out, genapi.Tag{Name: name, LinkCount: row.LinkCount, Curated: &curated})
		seen[name] = true
	}
	// Include curated tags even if no links currently use them.
	for _, t := range curatedRows {
		if seen[t.Name] {
			continue
		}
		curated := true
		out = append(out, genapi.Tag{Name: t.Name, LinkCount: 0, Curated: &curated})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	_, err := h.q.CreateUserTag(r.Context(), store.CreateUserTagParams{
		UserID: uuidToPgtype(uid),
		Name:   name,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "tag already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	curated := true
	writeJSON(w, http.StatusCreated, genapi.Tag{Name: name, LinkCount: 0, Curated: &curated})
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request, name string) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if err := h.q.DeleteUserTag(r.Context(), store.DeleteUserTagParams{
		UserID: uuidToPgtype(uid),
		Name:   name,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := h.q.GetUserByID(r.Context(), uuidToPgtype(uid))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusBadRequest, "invalid current password")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new password too short")
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if _, err := h.q.UpdateUserPassword(r.Context(), store.UpdateUserPasswordParams{
		ID:           uuidToPgtype(uid),
		PasswordHash: hash,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	u, err := h.q.GetUserSettings(r.Context(), uuidToPgtype(uid))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, genapi.UserSettings{
		RedirectDelaySeconds: int(u.RedirectDelaySeconds),
		KeepExpiredLinks:     u.KeepExpiredLinks,
		Timezone:             u.Timezone,
	})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req genapi.UpdateUserSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.RedirectDelaySeconds < 0 || req.RedirectDelaySeconds > 30 {
		writeError(w, http.StatusBadRequest, "redirect_delay_seconds out of range")
		return
	}
	u, err := h.q.UpdateUserSettings(r.Context(), store.UpdateUserSettingsParams{
		ID:                  uuidToPgtype(uid),
		RedirectDelaySeconds: int32(req.RedirectDelaySeconds),
		KeepExpiredLinks:     req.KeepExpiredLinks,
		Timezone:             req.Timezone,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, genapi.UserSettings{
		RedirectDelaySeconds: int(u.RedirectDelaySeconds),
		KeepExpiredLinks:     u.KeepExpiredLinks,
		Timezone:             u.Timezone,
	})
}

