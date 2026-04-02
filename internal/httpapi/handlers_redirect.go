package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

	// Optional interstitial with per-user customization.
	if link.UserID.Valid {
		if uid, ok := uuidFromPgtypeUUID(link.UserID); ok {
			if cfg, err := h.readRedirectCustomizationForUserID(r.Context(), uid); err == nil && r.URL.Query().Get("now") != "1" {
				if cfg.Mode == "click" || cfg.DelaySeconds > 0 {
					h.writeRedirectInterstitial(w, link.TargetUrl, cfg)
					return
				}
			}
		}
	}

	http.Redirect(w, r, link.TargetUrl, http.StatusFound)
}

func (h *Handler) readRedirectCustomizationForUserID(ctx context.Context, uid uuid.UUID) (RedirectCustomization, error) {
	row := h.db.QueryRow(ctx, `
SELECT redirect_delay_seconds, redirect_mode, redirect_show_screenshot, redirect_custom_buttons
FROM users
WHERE id = $1
`, uid)
	var out RedirectCustomization
	var raw []byte
	if err := row.Scan(&out.DelaySeconds, &out.Mode, &out.ShowScreenshot, &raw); err != nil {
		return RedirectCustomization{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.CustomButtons)
	}
	return normalizeRedirectCustomization(out), nil
}

func (h *Handler) screenshotPreviewSrc(target string) string {
	if tpl := strings.TrimSpace(h.cfg.ScreenshotPreviewURLTemplate); tpl != "" && strings.Contains(tpl, "{url}") {
		return strings.ReplaceAll(tpl, "{url}", url.QueryEscape(target))
	}
	if strings.TrimSpace(h.cfg.ScreenshotServiceURL) != "" {
		return strings.TrimRight(h.cfg.PublicBaseURL, "/") + "/preview/screenshot?url=" + url.QueryEscape(target)
	}
	return ""
}

func (h *Handler) writeRedirectInterstitial(w http.ResponseWriter, target string, cfg RedirectCustomization) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	mode := cfg.Mode
	if mode != "click" {
		mode = "auto"
	}
	delay := cfg.DelaySeconds
	if delay < 0 {
		delay = 0
	}
	if delay > 30 {
		delay = 30
	}
	targetEsc := htmlEscape(target)

	// Dark theme aligned with ShortWarden Vue app (#1c1f2a canvas, #242a3a card, lime accents).
	// Global <style> avoids a white flash before body paints; forced backgrounds match sw-shell.
	_, _ = w.Write([]byte(`<!doctype html><html lang="en" style="background:#1c1f2a!important"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><meta name="color-scheme" content="dark"/>`))
	_, _ = w.Write([]byte(`<title>Redirecting…</title><style>
:root{color-scheme:dark;}
html{background:#1c1f2a!important}
body{margin:0;min-height:100vh;font-family:system-ui,Segoe UI,sans-serif;line-height:1.5;background:#1c1f2a!important;color:#e2e8f0!important;-webkit-font-smoothing:antialiased}
*{box-sizing:border-box}
@keyframes sw-rd-spin{to{transform:rotate(360deg)}}
.sw-rd-spinner{width:40px;height:40px;border:3px solid rgba(255,255,255,0.12);border-top-color:#a3e635;border-radius:50%;animation:sw-rd-spin 0.75s linear infinite;flex-shrink:0}
</style></head>`))
	_, _ = w.Write([]byte(`<body style="background:#1c1f2a!important;color:#e2e8f0!important">`))
	_, _ = w.Write([]byte(`<div style="max-width:720px;margin:0 auto;padding:28px 24px 48px;">`))
	_, _ = w.Write([]byte(`<p style="margin:0 0 16px;font-size:11px;font-weight:600;letter-spacing:0.14em;text-transform:uppercase;color:#64748b">ShortWarden</p>`))
	_, _ = w.Write([]byte(`<div style="border-radius:16px;border:1px solid rgba(255,255,255,0.06);background:#242a3a;box-shadow:0 18px 60px rgba(0,0,0,0.55);padding:24px;">`))
	_, _ = w.Write([]byte(`<h1 style="font-size:18px;font-weight:600;margin:0 0 12px;color:#f1f5f9;letter-spacing:-0.02em;">Redirecting…</h1>`))

	if mode == "click" {
		if delay > 0 {
			_, _ = w.Write([]byte(`<p style="margin:0 0 12px;color:#cbd5e1;font-size:14px;">Click <strong style="color:#f1f5f9;">Continue</strong> when you are ready. Time remaining: <strong id="sec" style="color:#d9f99d;">` + strconv.Itoa(delay) + `</strong> s</p>`))
		} else {
			_, _ = w.Write([]byte(`<p style="margin:0 0 12px;color:#cbd5e1;font-size:14px;">Click <strong style="color:#f1f5f9;">Continue</strong> to open the destination.</p>`))
		}
	} else if delay > 0 {
		_, _ = w.Write([]byte(`<p style="margin:0 0 12px;color:#cbd5e1;font-size:14px;">You will be redirected in <strong id="sec" style="color:#d9f99d;">` + strconv.Itoa(delay) + `</strong> seconds.</p>`))
	} else {
		_, _ = w.Write([]byte(`<p style="margin:0 0 12px;color:#cbd5e1;font-size:14px;">Redirecting…</p>`))
	}

	_, _ = w.Write([]byte(`<p style="margin:0 0 14px;word-break:break-all;font-size:13px;"><a id="link" href="` + targetEsc + `" style="color:#bef264;text-decoration:underline;text-underline-offset:3px;">` + targetEsc + `</a></p>`))

	previewURL := ""
	if cfg.ShowScreenshot {
		previewURL = h.screenshotPreviewSrc(target)
		if previewURL != "" {
			_, _ = w.Write([]byte(`<div id="sw-preview-box" style="margin:0 0 12px;border-radius:12px;border:1px solid rgba(255,255,255,0.08);background:#141826;min-height:220px;position:relative;overflow:hidden">`))
			_, _ = w.Write([]byte(`<div id="sw-preview-loading" style="display:flex;flex-direction:column;align-items:center;justify-content:center;gap:14px;padding:36px 20px;min-height:220px">`))
			_, _ = w.Write([]byte(`<div class="sw-rd-spinner" aria-hidden="true"></div>`))
			_, _ = w.Write([]byte(`<div style="text-align:center"><p style="margin:0;font-size:14px;font-weight:600;color:#f1f5f9">Rendering preview…</p>`))
			_, _ = w.Write([]byte(`<p style="margin:8px 0 0;font-size:12px;color:#64748b;max-width:280px">Capturing the destination page. This can take a few seconds.</p></div></div>`))
			_, _ = w.Write([]byte(`<img id="sw-preview-img" alt="Page preview" decoding="async" referrerpolicy="no-referrer" src="` + htmlEscape(previewURL) + `" style="display:none;width:100%;vertical-align:middle;border-radius:12px;border:1px solid rgba(255,255,255,0.08)"/></div>`))
			_, _ = w.Write([]byte(`<p style="margin:0 0 14px;font-size:12px;color:#94a3b8">Preview from your ShortWarden screenshot service.</p>`))
		} else {
			_, _ = w.Write([]byte(`<p style="margin:0 0 14px;font-size:12px;color:#fcd34d;border-radius:10px;border:1px solid rgba(251,191,36,0.25);background:rgba(251,191,36,0.08);padding:10px 12px;">Screenshot preview is on, but the server is not configured. Set <code style="font-size:11px;color:#fde68a;">SHORTWARDEN_SCREENSHOT_SERVICE_URL</code> or a custom <code style="font-size:11px;color:#fde68a;">SHORTWARDEN_SCREENSHOT_PREVIEW_URL</code> with <code style="font-size:11px;color:#fde68a;">{url}</code>.</p>`))
		}
	}

	_, _ = w.Write([]byte(`<div style="display:flex;flex-wrap:wrap;gap:10px;margin-top:4px;"><a id="continue" href="` + targetEsc + `" style="display:inline-block;padding:10px 16px;border-radius:10px;background:#a3e635;color:#0f172a;font-weight:600;text-decoration:none;box-shadow:0 0 0 1px rgba(163,230,53,0.25),0 12px 32px rgba(163,230,53,0.12);">Continue</a>`))
	for _, b := range cfg.CustomButtons {
		_, _ = w.Write([]byte(`<a href="` + htmlEscape(b.Url) + `" style="display:inline-block;padding:10px 16px;border-radius:10px;border:1px solid rgba(255,255,255,0.1);background:rgba(255,255,255,0.05);color:#e2e8f0;text-decoration:none;font-weight:500;">` + htmlEscape(b.Label) + `</a>`))
	}
	_, _ = w.Write([]byte(`</div>`))

	_, _ = w.Write([]byte(`<script>`))
	_, _ = w.Write([]byte(`(function(){var mode="` + mode + `";var s=` + strconv.Itoa(delay) + `;var el=document.getElementById("sec");function tick(){if(el&&s>=0)el.textContent=s;}if(s<=0)return;tick();setInterval(function(){s--;tick();if(mode==="auto"&&s<=0){window.location.href=document.getElementById("link").href;}},1000);})();`))
	if previewURL != "" {
		_, _ = w.Write([]byte(`(function(){var img=document.getElementById("sw-preview-img");var box=document.getElementById("sw-preview-loading");if(!img||!box)return;function showImg(){box.style.display="none";img.style.display="block";}function showErr(){box.style.display="flex";box.style.flexDirection="column";box.style.alignItems="center";box.style.justifyContent="center";box.innerHTML="";var p=document.createElement("p");p.style.margin="0";p.style.padding="28px 16px";p.style.textAlign="center";p.style.fontSize="13px";p.style.color="#fcd34d";p.style.lineHeight="1.5";p.textContent="Preview could not be rendered.";box.appendChild(p);}img.addEventListener("load",showImg);img.addEventListener("error",showErr);if(img.complete&&img.naturalWidth>0)showImg();})();`))
	}
	_, _ = w.Write([]byte(`</script>`))
	base := strings.TrimRight(h.cfg.PublicBaseURL, "/")
	_, _ = w.Write([]byte(`<p style="margin-top:22px;padding-top:16px;border-top:1px solid rgba(255,255,255,0.08);font-size:12px;color:#64748b;">This short link is served by <a href="` + htmlEscape(base) + `" style="color:#bef264;text-decoration:none;font-weight:500;">ShortWarden</a>.</p>`))
	_, _ = w.Write([]byte(`</div></div></body></html>`))
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

