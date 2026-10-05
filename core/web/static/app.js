const listEl = document.getElementById("list");
const detailEl = document.getElementById("detail");
const form = document.getElementById("capture");
const rawEl = document.getElementById("raw");
const saveBtn = document.getElementById("save");
const saveStatus = document.getElementById("save-status");

let selectedId = null;
let pollTimer = null;

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const raw = rawEl.value.trim();
  if (!raw) return;
  saveBtn.disabled = true;
  saveStatus.textContent = "Saving…";
  try {
    const res = await fetch("/api/captures", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ raw }),
    });
    if (!res.ok) throw new Error(await res.text());
    const task = await res.json();
    rawEl.value = "";
    saveStatus.textContent = "Saved to disk.";
    selectedId = task.id;
    await refresh();
  } catch (err) {
    saveStatus.textContent = "Save failed: " + err.message;
  } finally {
    saveBtn.disabled = false;
  }
});

async function refresh() {
  const res = await fetch("/api/tasks");
  const tasks = await res.json();
  renderList(tasks);
  if (selectedId) {
    const t = tasks.find((x) => x.id === selectedId) || (await fetchTask(selectedId));
    if (t) renderDetail(t);
  }
}

async function fetchTask(id) {
  const res = await fetch("/api/tasks/" + encodeURIComponent(id));
  if (!res.ok) return null;
  return res.json();
}

function renderList(tasks) {
  listEl.innerHTML = "";
  if (!tasks.length) {
    listEl.innerHTML = "<li class='muted' style='padding:0.8rem 0'>Nothing captured yet.</li>";
    return;
  }
  for (const t of tasks) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "item" + (t.id === selectedId ? " active" : "");
    btn.innerHTML =
      `<span class="item-title">${escapeHtml(t.title || t.id)}</span>` +
      `<span class="item-meta">${badges(t)}</span>`;
    btn.addEventListener("click", () => {
      selectedId = t.id;
      renderDetail(t);
      refresh();
    });
    li.appendChild(btn);
    listEl.appendChild(li);
  }
}

function badges(t) {
  const bits = [`<span>${escapeHtml(t.status)}</span>`];
  if (t.needs_split) bits.push(`<span class="badge split">split?</span>`);
  if (t.enrichment === "failed") bits.push(`<span class="badge failed">enrich failed</span>`);
  if (t.enrichment === "pending") bits.push(`<span class="badge pending">enriching</span>`);
  if (t.enrichment === "ok") bits.push(`<span class="badge ok">enriched</span>`);
  return bits.join("");
}

function renderDetail(t) {
  const due = t.due_at || "—";
  const pri = t.priority || "—";
  const ctx = (t.context || []).join(", ") || "—";
  const qs = (t.open_questions || []).join(" · ") || "—";
  const splits = (t.split_candidates || [])
    .map((c) => `<li><strong>${escapeHtml(c.title)}</strong> — ${escapeHtml(c.excerpt || "")}</li>`)
    .join("");
  detailEl.className = "detail";
  detailEl.innerHTML = `
    <h2>${escapeHtml(t.title || t.id)}</h2>
    <div class="row" style="margin:0 0 1rem">
      <button type="button" id="retry-enrich">Retry enrich</button>
      <p class="status">${structuredStatus(t)}</p>
    </div>
    <dl class="kv">
      <dt>id</dt><dd>${escapeHtml(t.id)}</dd>
      <dt>status</dt><dd>${escapeHtml(t.status)}</dd>
      <dt>enrichment</dt><dd>${escapeHtml(t.enrichment)}</dd>
      <dt>requester</dt><dd>${escapeHtml(t.requester || "—")}</dd>
      <dt>due</dt><dd>${escapeHtml(due)}</dd>
      <dt>priority</dt><dd>${escapeHtml(pri)}</dd>
      <dt>context</dt><dd>${escapeHtml(ctx)}</dd>
      <dt>open</dt><dd>${escapeHtml(qs)}</dd>
    </dl>
    ${t.needs_split ? `<p class="badge split">Candidate for splitting</p><ul>${splits}</ul>` : ""}
    <h2>Raw</h2>
    <pre class="raw" dir="auto">${escapeHtml(t.raw)}</pre>
    <h2>Structured</h2>
    <pre class="structured" dir="auto">${escapeHtml(structuredBody(t))}</pre>
  `;
  const retry = document.getElementById("retry-enrich");
  if (retry) {
    retry.addEventListener("click", async () => {
      retry.disabled = true;
      await fetch("/api/tasks/" + encodeURIComponent(t.id) + "/enrich", { method: "POST" });
      const fresh = await fetchTask(t.id);
      if (fresh) renderDetail(fresh);
      refresh();
    });
  }
  clearTimeout(pollTimer);
  if (t.enrichment === "pending") {
    pollTimer = setTimeout(async () => {
      const fresh = await fetchTask(t.id);
      if (fresh) {
        renderDetail(fresh);
        refresh();
      }
    }, 1200);
  }
}

function structuredStatus(t) {
  if (t.enrichment === "pending") return "Enriching…";
  if (t.enrichment === "failed") return "Enrich failed" + (t.enrichment_error ? ": " + t.enrichment_error : "");
  return "Enriched";
}

function structuredBody(t) {
  if (t.structured) return t.structured;
  if (t.enrichment === "pending") return "Waiting on GPT…";
  if (t.enrichment === "failed") return t.enrichment_error || "Enrichment failed. Raw is safe on disk.";
  return "";
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

refresh().catch((err) => {
  saveStatus.textContent = "Could not load tasks: " + err.message;
});
