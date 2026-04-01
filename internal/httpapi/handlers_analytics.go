package httpapi

import (
	"errors"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/store"
)

func (h *Handler) GetLinkAnalytics(w http.ResponseWriter, r *http.Request, id genapi.IdParam, params genapi.GetLinkAnalyticsParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// Ensure ownership.
	_, err := h.q.GetLinkByID(r.Context(), store.GetLinkByIDParams{
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

	rows, err := h.q.ClickTotalsByDay(r.Context(), store.ClickTotalsByDayParams{
		LinkID:      uuidToPgtype(uuid.UUID(id)),
		ClickedAt:   pgtype.Timestamptz{Time: params.From.UTC(), Valid: true},
		ClickedAt_2: pgtype.Timestamptz{Time: params.To.UTC(), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	days := make([]genapi.LinkAnalyticsDay, 0, len(rows))
	for _, row := range rows {
		day := row.Day.Time
		days = append(days, genapi.LinkAnalyticsDay{
			Day:    day,
			Clicks: row.Clicks,
		})
	}

	writeJSON(w, http.StatusOK, genapi.LinkAnalytics{
		From: params.From.UTC(),
		To:   params.To.UTC(),
		Days: days,
	})
}

func (h *Handler) ListLinkClicks(w http.ResponseWriter, r *http.Request, id genapi.IdParam, params genapi.ListLinkClicksParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// Ensure ownership.
	_, err := h.q.GetLinkByID(r.Context(), store.GetLinkByIDParams{
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

	limit := 50
	offset := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}

	rows, err := h.q.ListClickEventsForLink(r.Context(), store.ListClickEventsForLinkParams{
		LinkID: uuidToPgtype(uuid.UUID(id)),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	out := make([]genapi.ClickEvent, 0, len(rows))
	for _, row := range rows {
		var ref *string
		if row.Referrer.Valid {
			ref = &row.Referrer.String
		}
		var ua *string
		if row.UserAgent.Valid {
			ua = &row.UserAgent.String
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
		out = append(out, genapi.ClickEvent{
			Id:        row.ID,
			ClickedAt: row.ClickedAt.Time,
			Referrer:  ref,
			UserAgent: ua,
			Ip:        ip,
			Country:   country,
			Device:    device,
		})
	}

	// Keep runtime/types imported in this file for consistency with others.
	var _ openapi_types.UUID
	var _ time.Time

	writeJSON(w, http.StatusOK, out)
}

