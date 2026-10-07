const listEl = document.getElementById("list");
const detailEl = document.getElementById("detail");
const pageStatus = document.getElementById("page-status");

let selectedId = null;
let orgCache = null;

async function loadOrg() {
  if (orgCache) return orgCache;
  try {
    const res = await fetch("/api/org");
    if (!res.ok) return { teams: [] };
    orgCache = await res.json();
    if (!orgCache.teams) orgCache.teams = [];
    return orgCache;
  } catch {
    return { teams: [] };
  }
}

async function refresh() {
  const res = await fetch("/api/tasks?archived=1");
  const tasks = await res.json();
  renderList(tasks);
  if (selectedId) {
    const t = tasks.find((x) => x.id === selectedId);
    if (t) renderDetail(t);
    else {
      selectedId = null;
      detailEl.className = "detail empty";
      detailEl.innerHTML = `<p class="muted">Select a task.</p>`;
    }
  }
}

function renderList(tasks) {
  listEl.innerHTML = "";
  if (!tasks.length) {
    listEl.innerHTML = "<li class='muted' style='padding:0.8rem 0'>Archive is empty.</li>";
    return;
  }
  for (const t of tasks) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "item" + (t.id === selectedId ? " active" : "");
    btn.innerHTML =
      `<span class="item-title" dir="auto">${escapeHtml(taskLabel(t))}</span>` +
      `<span class="item-meta"><span>${escapeHtml(t.status)}</span></span>`;
    btn.addEventListener("click", () => {
      selectedId = t.id;
      renderDetail(t);
      refresh();
    });
    li.appendChild(btn);
    listEl.appendChild(li);
  }
}

function taskLabel(t) {
  const line = String(t.raw || "")
    .replace(/\r\n/g, "\n")
    .split("\n")[0]
    .trim();
  if (!line) return t.id;
  return line.length > 80 ? line.slice(0, 80).trim() + "…" : line;
}

function renderDetail(t) {
  const qs = (t.open_questions || []).join(" · ") || "—";
  detailEl.className = "detail";
  detailEl.innerHTML = `
    <pre class="raw raw-top" dir="auto">${escapeHtml(t.raw || "")}</pre>
    <div class="row" style="margin:0 0 1rem">
      <button type="button" id="delete-task" class="danger">Delete</button>
    </div>
    <dl class="kv">
      <dt>id</dt><dd>${escapeHtml(t.id)}</dd>
      <dt>status</dt><dd>${escapeHtml(t.status)}</dd>
      <dt>requester</dt><dd dir="auto">${escapeHtml(t.requester || "—")}</dd>
      <dt>open</dt><dd dir="auto">${escapeHtml(qs)}</dd>
      <dt>teams</dt><dd id="rel-teams">…</dd>
      <dt>people</dt><dd id="rel-people">…</dd>
    </dl>
  `;
  fillRelated(t);
  const del = document.getElementById("delete-task");
  if (del) {
    del.addEventListener("click", async () => {
      if (!confirm("Delete this archived task from disk?")) return;
      del.disabled = true;
      const done = memoryHold();
      try {
        const res = await fetch("/api/tasks/" + encodeURIComponent(t.id), { method: "DELETE" });
        if (!res.ok) throw new Error(await res.text());
        selectedId = null;
        pageStatus.textContent = "";
        await refresh();
        detailEl.className = "detail empty";
        detailEl.innerHTML = `<p class="muted">Select a task.</p>`;
      } catch (err) {
        pageStatus.textContent = "Delete failed: " + err.message;
        del.disabled = false;
      } finally {
        done();
      }
    });
  }
}

async function fillRelated(t) {
  const teamsEl = document.getElementById("rel-teams");
  const peopleEl = document.getElementById("rel-people");
  if (!teamsEl || !peopleEl) return;
  const org = await loadOrg();
  const teamNames = new Map();
  const empNames = new Map();
  for (const team of org.teams || []) {
    teamNames.set(team.id, team.name);
    for (const e of team.employees || []) {
      empNames.set(e.id, e.name);
    }
  }
  const teams = (t.related_teams || []).map((id) => teamNames.get(id) || id);
  const people = (t.related_employees || []).map((id) => empNames.get(id) || id);
  teamsEl.textContent = teams.length ? teams.join(", ") : "—";
  peopleEl.textContent = people.length ? people.join(", ") : "—";
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

Promise.all([loadOrg(), refresh()]).catch((err) => {
  pageStatus.textContent = "Could not load archive: " + err.message;
});
