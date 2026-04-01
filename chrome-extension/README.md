# ShortWarden Chrome extension

## Install (developer mode)

1. Chrome → `chrome://extensions`
2. Enable **Developer mode**
3. Click **Load unpacked**
4. Select this folder: `chrome-extension/`

## Configure

1. In the ShortWarden web UI create an API key under `/app/api-keys`
2. Extension → **Details** → **Extension options**
3. Set:
   - API base URL (e.g. `http://localhost:8080`)
   - API key (starts with `sw_...`)

## Use

- Click the extension icon → “Create short link” (copies the result to clipboard)
- Right click page → “Shorten this page”

