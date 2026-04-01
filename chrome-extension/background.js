chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: "shortwarden_shorten_page",
    title: "Shorten this page",
    contexts: ["page"],
  });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId !== "shortwarden_shorten_page") return;
  const url = tab?.url || info.pageUrl;
  if (!url) return;

  const { baseUrl, apiKey } = await chrome.storage.local.get(["baseUrl", "apiKey"]);
  const apiBase = (baseUrl || "http://localhost:8080").replace(/\/$/, "");
  if (!apiKey) {
    return;
  }

  try {
    const res = await fetch(`${apiBase}/v1/links`, {
      method: "POST",
      headers: { "content-type": "application/json", Authorization: `ApiKey ${apiKey}` },
      body: JSON.stringify({ target_url: url }),
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    const link = await res.json();
    const shortUrl = link.short_url || `${apiBase}/r/${link.alias}`;
    await chrome.scripting?.executeScript({
      target: { tabId: tab.id },
      func: async (text) => {
        await navigator.clipboard.writeText(text);
      },
      args: [shortUrl],
    });
  } catch {
    // ignore; popup has better UX
  }
});

