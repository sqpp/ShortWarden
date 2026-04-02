async function load() {
  const { baseUrl, apiKey } = await chrome.storage.local.get(["baseUrl", "apiKey"]);
  document.getElementById("baseUrl").value = baseUrl || "http://localhost:8080";
  document.getElementById("apiKey").value = apiKey || "";
}

function loadAbout() {
  try {
    const m = chrome.runtime.getManifest();
    document.getElementById("ver").textContent = m.version || "—";
  } catch {
    document.getElementById("ver").textContent = "—";
  }
  try {
    document.getElementById("extId").textContent = chrome.runtime.id || "—";
  } catch {
    document.getElementById("extId").textContent = "—";
  }
}

async function save() {
  const baseUrl = document.getElementById("baseUrl").value.trim();
  const apiKey = document.getElementById("apiKey").value.trim();
  await chrome.storage.local.set({ baseUrl, apiKey });
  setMsg("Saved.");
  setStatus("Saved", true);
}

async function test() {
  const { baseUrl, apiKey } = await chrome.storage.local.get(["baseUrl", "apiKey"]);
  if (!baseUrl || !apiKey) {
    setMsg("Set baseUrl and apiKey first.");
    setStatus("Missing values", false);
    return;
  }
  try {
    const res = await fetch(`${baseUrl.replace(/\/$/, "")}/v1/me`, {
      headers: { Authorization: `ApiKey ${apiKey}` },
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}: ${await res.text()}`);
    const me = await res.json();
    setMsg(`OK. Signed in as: ${me.email}`);
    setStatus("OK", true);
  } catch (e) {
    setMsg(`Test failed: ${e instanceof Error ? e.message : String(e)}`);
    setStatus("Test failed", false);
  }
}

function setMsg(s) {
  document.getElementById("msg").textContent = s;
}

function setStatus(text, ok) {
  const el = document.getElementById("status");
  el.textContent = text;
  el.style.borderColor = ok ? "rgba(163,230,53,.25)" : "rgba(239,68,68,.35)";
  el.style.background = ok ? "rgba(163,230,53,.08)" : "rgba(127,29,29,.28)";
  el.style.color = ok ? "#d9f99d" : "#fecaca";
}

document.getElementById("save").addEventListener("click", save);
document.getElementById("test").addEventListener("click", test);
loadAbout();
load();

