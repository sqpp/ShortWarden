async function load() {
  const { baseUrl, apiKey } = await chrome.storage.local.get(["baseUrl", "apiKey"]);
  document.getElementById("baseUrl").value = baseUrl || "http://localhost:8080";
  document.getElementById("apiKey").value = apiKey || "";
}

async function save() {
  const baseUrl = document.getElementById("baseUrl").value.trim();
  const apiKey = document.getElementById("apiKey").value.trim();
  await chrome.storage.local.set({ baseUrl, apiKey });
  setMsg("Saved.");
}

async function test() {
  const { baseUrl, apiKey } = await chrome.storage.local.get(["baseUrl", "apiKey"]);
  if (!baseUrl || !apiKey) {
    setMsg("Set baseUrl and apiKey first.");
    return;
  }
  try {
    const res = await fetch(`${baseUrl.replace(/\/$/, "")}/v1/me`, {
      headers: { Authorization: `ApiKey ${apiKey}` },
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}: ${await res.text()}`);
    const me = await res.json();
    setMsg(`OK. Signed in as: ${me.email}`);
  } catch (e) {
    setMsg(`Test failed: ${e instanceof Error ? e.message : String(e)}`);
  }
}

function setMsg(s) {
  document.getElementById("msg").textContent = s;
}

document.getElementById("save").addEventListener("click", save);
document.getElementById("test").addEventListener("click", test);
load();

