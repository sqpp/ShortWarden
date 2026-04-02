package httpapi

import (
	"net/http"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/store"
)

func (h *Handler) ListRecentClicks(w http.ResponseWriter, r *http.Request, params genapi.ListRecentClicksParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}

	rows, err := h.q.ListRecentClicks(r.Context(), store.ListRecentClicksParams{
		UserID: uuidToPgtype(uid),
		Limit:  int32(limit),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	out := make([]genapi.RecentClick, 0, len(rows))
	for _, row := range rows {
		lid, _ := uuidFromPgtypeUUID(row.LinkID)
		var did *openapi_types.UUID
		if row.DomainID.Valid {
			d, _ := uuidFromPgtypeUUID(row.DomainID)
			tmp := openapi_types.UUID(d)
			did = &tmp
		}
		var ref *string
		if row.Referrer.Valid {
			ref = &row.Referrer.String
		}
		var ip *string
		if row.Ip != nil {
			s := row.Ip.String()
			ip = &s
		}
		var country *string
		if row.Country.Valid {
			country = &row.Country.String
		}
		var device *string
		if row.Device.Valid {
			device = &row.Device.String
		}

		// Build short_url via existing helper.
		link := store.Link{
			ID:       row.LinkID,
			UserID:   uuidToPgtype(uid),
			DomainID: row.DomainID,
			Alias:    row.Alias,
		}
		short := h.linkToAPI(r.Context(), requestScheme(r), uid, link).ShortUrl

		out = append(out, genapi.RecentClick{
			Id:        row.ID,
			ClickedAt: row.ClickedAt.Time,
			LinkId:    openapi_types.UUID(lid),
			Alias:     row.Alias,
			DomainId:  did,
			Referrer:  ref,
			Ip:        ip,
			Country:   country,
			Device:    device,
			ShortUrl:  short,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) ListTopLinks(w http.ResponseWriter, r *http.Request, params genapi.ListTopLinksParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	limit := 10
	days := 7
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Days != nil {
		days = *params.Days
	}

	rows, err := h.q.ListTopLinksByClicks(r.Context(), store.ListTopLinksByClicksParams{
		UserID: uuidToPgtype(uid),
		Limit:  int32(limit),
		Column3: int32(days),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	out := make([]genapi.TopLink, 0, len(rows))
	for _, row := range rows {
		l := store.Link{
			ID:        row.ID,
			UserID:    row.UserID,
			DomainID:  row.DomainID,
			Alias:     row.Alias,
			TargetUrl: row.TargetUrl,
			Title:     row.Title,
			Tags:      row.Tags,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			ExpiresAt: row.ExpiresAt,
			DeletedAt: row.DeletedAt,
		}
		apiLink := h.linkToAPIWithClickCount(r.Context(), requestScheme(r), uid, l, row.Clicks)
		out = append(out, genapi.TopLink{Link: apiLink, Clicks: row.Clicks})
	}
	writeJSON(w, http.StatusOK, out)

	// keep imports stable
	_ = strings.TrimSpace
	_ = uuid.UUID{}
	_ = time.Time{}
}

