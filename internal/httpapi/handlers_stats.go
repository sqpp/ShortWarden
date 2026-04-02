package httpapi

import (
	"net/http"

	genapi "shortwarden/api/gen"
)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	row, err := h.q.GetUserStats(r.Context(), uuidToPgtype(uid))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, genapi.Stats{
		LinksTotal:  row.LinksTotal,
		ClicksTotal: row.ClicksTotal,
		Clicks24h:   row.Clicks24h,
		Clicks7d:    row.Clicks7d,
	})
}

