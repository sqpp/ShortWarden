package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	genapi "shortwarden/api/gen"
	"shortwarden/internal/auth"
	"shortwarden/internal/store"
)

func (h *Handler) ExportLinks(w http.ResponseWriter, r *http.Request, params genapi.ExportLinksParams) {
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	format := ""
	if params.Format != nil {
		format = string(*params.Format)
	} else {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/csv") {
			format = "csv"
		} else {
			format = "json"
		}
	}

	// Pull all non-deleted links for now (cap to a large limit).
	rows, err := h.q.ListLinks(r.Context(), store.ListLinksParams{
		UserID:   uuidToPgtype(uid),
		Limit:    5000,
		Column3:  false,
		Offset:   0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	exports := make([]genapi.LinkExport, 0, len(rows))
	for _, l := range rows {
		exports = append(exports, h.linkExportFor(r.Context(), uid, l))
	}

	if format == "csv" {
		writeCSVExports(w, exports)
		return
	}
	writeJSON(w, http.StatusOK, exports)
}

func writeCSVExports(w http.ResponseWriter, items []genapi.LinkExport) {
	w.Header().Set("content-type", "text/csv; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"domain_hostname", "alias", "target_url", "expires_at", "title", "tags"})
	for _, it := range items {
		dh := ""
		if it.DomainHostname != nil {
			dh = *it.DomainHostname
		}
		exp := ""
		if it.ExpiresAt != nil {
			exp = it.ExpiresAt.UTC().Format(time.RFC3339)
		}
		title := ""
		if it.Title != nil {
			title = *it.Title
		}
		tags := ""
		if it.Tags != nil {
			tags = strings.Join(*it.Tags, "|")
		}
		_ = cw.Write([]string{dh, it.Alias, it.TargetUrl, exp, title, tags})
	}
	cw.Flush()
}

func (h *Handler) ImportLinks(w http.ResponseWriter, r *http.Request, params genapi.ImportLinksParams) {
	if !requireCookieCSRF(w, r) {
		return
	}
	uid, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	dryRun := params.DryRun != nil && *params.DryRun

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	ct := r.Header.Get("Content-Type")
	ct = strings.Split(ct, ";")[0]

	var items []genapi.LinkImport
	switch ct {
	case "text/csv":
		items, err = parseImportCSV(body)
	default:
		// JSON by default.
		if err := json.Unmarshal(body, &items); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created := 0
	skipped := 0
	errs := make([]string, 0)

	for i, it := range items {
		if strings.TrimSpace(it.Alias) == "" || !aliasRe.MatchString(it.Alias) {
			errs = append(errs, "row "+itoa(i)+": invalid alias")
			continue
		}
		if !isValidURL(it.TargetUrl) {
			errs = append(errs, "row "+itoa(i)+": invalid target_url")
			continue
		}

		domainID := pgtype.UUID{Valid: false}
		if it.DomainHostname != nil && strings.TrimSpace(*it.DomainHostname) != "" {
			hostname, err := normalizeHostname(*it.DomainHostname)
			if err != nil {
				errs = append(errs, "row "+itoa(i)+": invalid domain_hostname")
				continue
			}
			d, err := h.q.GetDomainByHostname(r.Context(), store.GetDomainByHostnameParams{
				UserID:   uuidToPgtype(uid),
				Hostname: hostname,
			})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					raw, err := auth.NewRawAPIKey()
					if err != nil {
						errs = append(errs, "row "+itoa(i)+": token error")
						continue
					}
					token := strings.TrimPrefix(raw, "sw_")
					d, err = h.q.CreateDomain(r.Context(), store.CreateDomainParams{
						UserID:   uuidToPgtype(uid),
						Hostname: hostname,
						DnsToken: token,
					})
					if err != nil {
						errs = append(errs, "row "+itoa(i)+": domain create failed")
						continue
					}
				} else {
					errs = append(errs, "row "+itoa(i)+": domain lookup failed")
					continue
				}
			}
			domainID = d.ID
		}

		var expires pgtype.Timestamptz
		if it.ExpiresAt != nil {
			expires = pgtype.Timestamptz{Time: it.ExpiresAt.UTC(), Valid: true}
		}
		var title pgtype.Text
		if it.Title != nil && strings.TrimSpace(*it.Title) != "" {
			title = pgtype.Text{String: *it.Title, Valid: true}
		}
		tags := []string{}
		if it.Tags != nil {
			tags = *it.Tags
		}

		if dryRun {
			created++
			continue
		}

		_, err = h.q.CreateLink(r.Context(), store.CreateLinkParams{
			UserID:    uuidToPgtype(uid),
			DomainID:  domainID,
			Alias:     it.Alias,
			TargetUrl: it.TargetUrl,
			Title:     title,
			Tags:      tags,
			ExpiresAt: expires,
		})
		if err != nil {
			if isUniqueViolation(err) {
				skipped++
				continue
			}
			errs = append(errs, "row "+itoa(i)+": create failed")
			continue
		}
		created++
	}

	writeJSON(w, http.StatusOK, genapi.ImportResult{
		Created: created,
		Skipped: skipped,
		Errors:  errs,
	})
}

func parseImportCSV(b []byte) ([]genapi.LinkImport, error) {
	cr := csv.NewReader(bytes.NewReader(b))
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, errors.New("invalid csv")
	}
	if len(rows) == 0 {
		return []genapi.LinkImport{}, nil
	}
	// Expect header: domain_hostname,alias,target_url,expires_at,title,tags
	start := 0
	if len(rows[0]) > 0 && strings.EqualFold(strings.TrimSpace(rows[0][0]), "domain_hostname") {
		start = 1
	}
	out := make([]genapi.LinkImport, 0, len(rows)-start)
	for _, r := range rows[start:] {
		get := func(i int) string {
			if i >= 0 && i < len(r) {
				return strings.TrimSpace(r[i])
			}
			return ""
		}
		var dh *string
		if v := get(0); v != "" {
			dh = &v
		}
		alias := get(1)
		target := get(2)
		var exp *time.Time
		if v := get(3); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				tt := t.UTC()
				exp = &tt
			}
		}
		var title *string
		if v := get(4); v != "" {
			title = &v
		}
		var tags *[]string
		if v := get(5); v != "" {
			parts := strings.Split(v, "|")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			tags = &parts
		}
		out = append(out, genapi.LinkImport{
			DomainHostname: dh,
			Alias:          alias,
			TargetUrl:      target,
			ExpiresAt:      exp,
			Title:          title,
			Tags:           tags,
		})
	}
	return out, nil
}

func (h *Handler) linkExportFor(ctx context.Context, uid uuid.UUID, l store.Link) genapi.LinkExport {
	var domainHostname *string
	if l.DomainID.Valid {
		if d, err := h.q.GetDomainByID(ctx, store.GetDomainByIDParams{
			ID:     l.DomainID,
			UserID: uuidToPgtype(uid),
		}); err == nil {
			domainHostname = &d.Hostname
		}
	}
	var exp *time.Time
	if l.ExpiresAt.Valid {
		exp = &l.ExpiresAt.Time
	}
	var title *string
	if l.Title.Valid {
		title = &l.Title.String
	}
	tags := l.Tags
	return genapi.LinkExport{
		DomainHostname: domainHostname,
		Alias:          l.Alias,
		TargetUrl:      l.TargetUrl,
		ExpiresAt:      exp,
		Title:          title,
		Tags:           &tags,
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

