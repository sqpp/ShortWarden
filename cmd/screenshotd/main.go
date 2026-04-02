package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"shortwarden/internal/screenshoturl"
)

func main() {
	addr := envOr("SCREENSHOTD_HTTP_ADDR", ":8088")
	chromePath := envOr("CHROMIUM_PATH", "/usr/bin/chromium")

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		handleRender(w, r, chromePath)
	})

	log.Printf("screenshotd listening on %s (chromium=%s)", addr, chromePath)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func handleRender(w http.ResponseWriter, r *http.Request, chromePath string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	u, err := screenshoturl.ValidateTargetURL(raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	target := u.String()
	png, err := capture(r.Context(), chromePath, target)
	if err != nil {
		log.Printf("capture %s: %v", target, err)
		http.Error(w, "capture failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func capture(ctx context.Context, chromePath, targetURL string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.WindowSize(1280, 800),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()
	ctx, cancel := context.WithTimeout(taskCtx, 25*time.Second)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(800*time.Millisecond),
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		return nil, fmt.Errorf("chromedp: %w", err)
	}
	return buf, nil
}
