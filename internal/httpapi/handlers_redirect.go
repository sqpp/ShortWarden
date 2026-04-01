package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"shortwarden/internal/store"
)

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request, alias string) {
	link, err := h.q.GetActiveLinkByAlias(r.Context(), alias)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	// Fire-and-forget click tracking; don't block the redirect.
	ref := strings.TrimSpace(r.Referer())
	ua := strings.TrimSpace(r.UserAgent())
	remoteAddr := r.RemoteAddr
	go h.recordClick(link.ID, ref, ua, remoteAddr)

	// Optional interstitial with countdown (per-user setting).
	if link.UserID.Valid {
		if uid, ok := uuidFromPgtypeUUID(link.UserID); ok {
			if u, err := h.q.GetUserSettings(r.Context(), uuidToPgtype(uid)); err == nil {
				if u.RedirectDelaySeconds > 0 && u.RedirectDelaySeconds <= 30 && r.URL.Query().Get("now") != "1" {
					writeRedirectInterstitial(w, link.TargetUrl, int(u.RedirectDelaySeconds))
					return
				}
			}
		}
	}

	http.Redirect(w, r, link.TargetUrl, http.StatusFound)
}

func writeRedirectInterstitial(w http.ResponseWriter, target string, seconds int) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Minimal HTML to avoid CSP issues; works behind nginx too.
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/>`))
	_, _ = w.Write([]byte(`<title>Redirecting…</title>`))
	_, _ = w.Write([]byte(`</head><body style="font-family:system-ui,Segoe UI,sans-serif;padding:24px;max-width:720px;margin:0 auto;">`))
	_, _ = w.Write([]byte(`<h1 style="font-size:18px;margin:0 0 8px;">Redirecting…</h1>`))
	_, _ = w.Write([]byte(`<p style="margin:0 0 12px;">You will be redirected in <strong id="sec"></strong> seconds.</p>`))
	_, _ = w.Write([]byte(`<p style="margin:0 0 12px;word-break:break-all;"><a id="link" href="` + htmlEscape(target) + `">` + htmlEscape(target) + `</a></p>`))
	_, _ = w.Write([]byte(`<script>
var s = ` + strconv.Itoa(seconds) + `;
var el = document.getElementById('sec');
el.textContent = s;
var t = setInterval(function(){
  s--;
  el.textContent = s;
  if(s<=0){ clearInterval(t); window.location.href = document.getElementById('link').href; }
}, 1000);
</script>`))
	_, _ = w.Write([]byte(`</body></html>`))
}

func htmlEscape(s string) string {
	// small escape to avoid pulling html/template for now
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}

func (h *Handler) recordClick(linkID pgtype.UUID, ref, ua, remoteAddr string) {
	// Use a short-lived background context; request context may be canceled immediately after redirect.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var refT pgtype.Text
	if ref != "" {
		refT = pgtype.Text{String: ref, Valid: true}
	}
	var uaT pgtype.Text
	if ua != "" {
		uaT = pgtype.Text{String: ua, Valid: true}
	}

	var ip *netip.Addr
	// Best-effort. RealIP middleware will set r.RemoteAddr accordingly if configured behind proxies.
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			ip = &addr
		}
	}

	_, _ = h.q.InsertClickEvent(ctx, store.InsertClickEventParams{
		LinkID:    linkID,
		Referrer:  refT,
		UserAgent: uaT,
		Ip:        ip,
		Country:   pgtype.Text{},
		Device:    pgtype.Text{},
	})
}

