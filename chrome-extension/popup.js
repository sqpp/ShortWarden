async function getActiveTabUrl() {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  return tabs[0]?.url || "";
}

async function getConfig() {
  const { baseUrl, apiKey } = await chrome.storage.local.get(["baseUrl", "apiKey"]);
  return {
    baseUrl: (baseUrl || "http://localhost:8080").replace(/\/$/, ""),
    apiKey: apiKey || "",
  };
}

function setMsg(s) {
  document.getElementById("msg").textContent = s;
}

function computeExpires(preset, customValue) {
  if (!preset) return "";
  if (preset === "custom") return customValue.trim();
  const now = new Date();
  const add = (ms) => new Date(now.getTime() + ms).toISOString();
  switch (preset) {
    case "5m":
      return add(5 * 60 * 1000);
    case "15m":
      return add(15 * 60 * 1000);
    case "30m":
      return add(30 * 60 * 1000);
    case "1h":
      return add(60 * 60 * 1000);
    case "6h":
      return add(6 * 60 * 60 * 1000);
    case "24h":
      return add(24 * 60 * 60 * 1000);
    case "3d":
      return add(3 * 24 * 60 * 60 * 1000);
    case "7d":
      return add(7 * 24 * 60 * 60 * 1000);
    case "1mo": {
      const d = new Date(now);
      d.setMonth(d.getMonth() + 1);
      return d.toISOString();
    }
    default:
      return "";
  }
}

async function shorten() {
  setMsg("");
  const { baseUrl, apiKey } = await getConfig();
  if (!apiKey) {
    setMsg("Missing API key. Open Options and paste one.");
    return;
  }

  const url = document.getElementById("url").value.trim();
  const alias = document.getElementById("alias").value.trim();
  const preset = document.getElementById("expiryPreset").value;
  const expires = computeExpires(preset, document.getElementById("expires").value);

  const payload = { target_url: url };
  if (alias) payload.alias = alias;
  if (expires) payload.expires_at = expires;

  try {
    const res = await fetch(`${baseUrl}/v1/links`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        Authorization: `ApiKey ${apiKey}`,
      },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}: ${await res.text()}`);
    const link = await res.json();
    const shortUrl = link.short_url || `${baseUrl}/r/${link.alias}`;
    await navigator.clipboard.writeText(shortUrl);
    setMsg(`Created:\n${shortUrl}\n\nCopied to clipboard.`);
  } catch (e) {
    setMsg(`Failed: ${e instanceof Error ? e.message : String(e)}`);
  }
}

document.getElementById("shorten").addEventListener("click", shorten);
document.getElementById("openApp").addEventListener("click", async () => {
  const { baseUrl } = await getConfig();
  chrome.tabs.create({ url: `${baseUrl}/` });
});
document.getElementById("openOptions").addEventListener("click", () => chrome.runtime.openOptionsPage());

(async () => {
  document.getElementById("url").value = await getActiveTabUrl();
  const presetEl = document.getElementById("expiryPreset");
  const customEl = document.getElementById("expires");
  presetEl.addEventListener("change", () => {
    customEl.style.display = presetEl.value === "custom" ? "block" : "none";
  });
})();

