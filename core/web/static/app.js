const listEl = document.getElementById("list");
const detailEl = document.getElementById("detail");
const form = document.getElementById("capture");
const rawEl = document.getElementById("raw");
const saveBtn = document.getElementById("save");
const saveStatus = document.getElementById("save-status");
const newBtn = document.getElementById("new-task");
const tabInbox = document.getElementById("tab-inbox");
const tabArchive = document.getElementById("tab-archive");

let selectedId = null;
let listMode = "inbox";
let pollTimer = null;
let orgCache = null;
let storiesCache = null;

function readListParam() {
  const v = new URLSearchParams(location.search).get("list");
  return v === "archive" ? "archive" : "inbox";
}

function setListMode(mode, { push = true } = {}) {
  listMode = mode === "archive" ? "archive" : "inbox";
  tabInbox.setAttribute("aria-selected", listMode === "inbox" ? "true" : "false");
  tabArchive.setAttribute("aria-selected", listMode === "archive" ? "true" : "false");
  newBtn.hidden = listMode !== "inbox";
  if (push) {
    const url = listMode === "archive" ? "/?list=archive" : "/";
    history.replaceState(null, "", url);
  }
  selectedId = null;
  showComposer();
  refresh();
}

function showComposer() {
  selectedId = null;
  clearTimeout(pollTimer);
  if (listMode === "inbox") {
    form.hidden = false;
    detailEl.hidden = true;
    detailEl.innerHTML = "";
    rawEl.focus();
  } else {
    form.hidden = true;
    detailEl.hidden = false;
    detailEl.innerHTML = "";
  }
  renderListActive();
}

function showDetail() {
  form.hidden = true;
  detailEl.hidden = false;
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const raw = rawEl.value.trim();
  if (!raw) return;
  saveBtn.disabled = true;
  const done = memoryHold();
  try {
    const res = await fetch("/api/captures", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ raw }),
    });
    if (!res.ok) throw new Error(await res.text());
    const task = await res.json();
    rawEl.value = "";
    saveStatus.textContent = "";
    selectedId = task.id;
    if (listMode !== "inbox") {
      listMode = "inbox";
      tabInbox.setAttribute("aria-selected", "true");
      tabArchive.setAttribute("aria-selected", "false");
      newBtn.hidden = false;
      history.replaceState(null, "", "/");
    }
    await refresh();
  } catch (err) {
    saveStatus.textContent = err.message;
  } finally {
    done();
    saveBtn.disabled = false;
  }
});

newBtn.addEventListener("click", () => {
  if (listMode !== "inbox") setListMode("inbox");
  else showComposer();
});
tabInbox.addEventListener("click", () => setListMode("inbox"));
tabArchive.addEventListener("click", () => setListMode("archive"));

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

async function loadStories() {
  if (storiesCache) return storiesCache;
  try {
    const res = await fetch("/api/stories");
    if (!res.ok) return { stories: [] };
    storiesCache = await res.json();
    if (!storiesCache.stories) storiesCache.stories = [];
    return storiesCache;
  } catch {
    return { stories: [] };
  }
}

async function refresh() {
  const q = listMode === "archive" ? "archived=1" : "archived=0";
  const res = await fetch("/api/tasks?" + q);
  const tasks = await res.json();
  renderList(tasks);
  if (selectedId) {
    const t = tasks.find((x) => x.id === selectedId);
    if (t) renderDetail(t);
    else showComposer();
  }
}

async function fetchTask(id) {
  const res = await fetch("/api/tasks/" + encodeURIComponent(id));
  if (!res.ok) return null;
  return res.json();
}

function renderListActive() {
  listEl.querySelectorAll("button.item").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.id === selectedId);
  });
}

function renderList(tasks) {
  listEl.innerHTML = "";
  for (const t of tasks) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "item" + (t.id === selectedId ? " active" : "");
    btn.dataset.id = t.id;
    btn.innerHTML =
      `<span class="item-title" dir="auto">${escapeHtml(taskLabel(t))}</span>` +
      `<span class="item-meta">${badges(t)}</span>`;
    btn.addEventListener("click", () => {
      selectedId = t.id;
      renderDetail(t);
      renderListActive();
    });
    li.appendChild(btn);
    listEl.appendChild(li);
  }
}

function badges(t) {
  const parts = [`<span>${escapeHtml(t.status)}</span>`];
  if (listMode === "inbox") {
    if (t.fill === "pending") parts.push(`<span class="badge pending">fill</span>`);
    if (t.fill === "failed") parts.push(`<span class="badge failed">fill</span>`);
  }
  return parts.join("");
}

function taskLabel(t) {
  const line = String(t.raw || "")
    .replace(/\r\n/g, "\n")
    .split("\n")[0]
    .trim();
  if (!line) return t.id;
  return line.length > 80 ? line.slice(0, 80).trim() + "…" : line;
}

const refreshIcon = `<svg class="icon-refresh" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M17.65 6.35A7.95 7.95 0 0 0 12 4a8 8 0 1 0 7.75 10h-2.1A6 6 0 1 1 12 6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg>`;

function renderDetail(t) {
  const qs = (t.open_questions || []).join(" · ") || "—";
  const filled = t.fill === "ok";
  const archived = listMode === "archive" || t.archived;
  showDetail();
  detailEl.innerHTML = `
    <div class="detail-top">
      <pre class="raw raw-top" dir="auto">${escapeHtml(t.raw || "")}</pre>
      ${
        archived
          ? ""
          : `<button type="button" id="retry-fill" class="icon-btn ${filled ? "filled" : ""}" title="Refresh fill" aria-label="Refresh fill">${refreshIcon}</button>`
      }
    </div>
    <div class="row" style="margin:0 0 1rem">
      ${
        archived
          ? `<button type="button" id="delete-task" class="danger">Delete</button>`
          : `<button type="button" id="archive-task" class="ghost">Archive</button>`
      }
    </div>
    <dl class="kv">
      <dt>id</dt><dd>${escapeHtml(t.id)}</dd>
      <dt>status</dt><dd>${escapeHtml(t.status)}</dd>
      <dt>requester</dt><dd dir="auto">${escapeHtml(t.requester || "—")}</dd>
      <dt>open</dt><dd dir="auto">${escapeHtml(qs)}</dd>
      <dt>teams</dt><dd><div class="chip-row" id="rel-teams"></div></dd>
      <dt>people</dt><dd><div class="chip-row" id="rel-people"></div></dd>
      <dt>story</dt><dd><div class="chip-row" id="rel-story"></div></dd>
    </dl>
  `;
  renderRelations(t);

  const retry = document.getElementById("retry-fill");
  if (retry) {
    retry.addEventListener("click", async () => {
      retry.disabled = true;
      orgCache = null;
      storiesCache = null;
      await fetch("/api/tasks/" + encodeURIComponent(t.id) + "/fill", { method: "POST" });
      const fresh = await fetchTask(t.id);
      if (fresh) renderDetail(fresh);
      refresh();
    });
  }

  const archiveBtn = document.getElementById("archive-task");
  if (archiveBtn) {
    archiveBtn.addEventListener("click", async () => {
      archiveBtn.disabled = true;
      const done = memoryHold();
      try {
        const res = await fetch("/api/tasks/" + encodeURIComponent(t.id) + "/archive", { method: "POST" });
        if (!res.ok) throw new Error(await res.text());
        saveStatus.textContent = "";
        showComposer();
        await refresh();
      } catch (err) {
        saveStatus.textContent = err.message;
        archiveBtn.disabled = false;
      } finally {
        done();
      }
    });
  }

  const del = document.getElementById("delete-task");
  if (del) {
    del.addEventListener("click", async () => {
      if (!confirm("Delete this archived task from disk?")) return;
      del.disabled = true;
      const done = memoryHold();
      try {
        const res = await fetch("/api/tasks/" + encodeURIComponent(t.id), { method: "DELETE" });
        if (!res.ok) throw new Error(await res.text());
        saveStatus.textContent = "";
        showComposer();
        await refresh();
      } catch (err) {
        saveStatus.textContent = err.message;
        del.disabled = false;
      } finally {
        done();
      }
    });
  }

  clearTimeout(pollTimer);
  if (!archived && t.fill === "pending") {
    pollTimer = setTimeout(async () => {
      const fresh = await fetchTask(t.id);
      if (fresh && !fresh.archived) {
        renderDetail(fresh);
        refresh();
      }
    }, 1200);
  }
}

async function renderRelations(t) {
  const teamsEl = document.getElementById("rel-teams");
  const peopleEl = document.getElementById("rel-people");
  const storyEl = document.getElementById("rel-story");
  if (!teamsEl || !peopleEl || !storyEl) return;
  const [org, storyList] = await Promise.all([loadOrg(), loadStories()]);
  const teamOpts = (org.teams || []).map((team) => ({ id: team.id, label: team.name }));
  const peopleOpts = [];
  for (const team of org.teams || []) {
    for (const e of team.employees || []) {
      peopleOpts.push({ id: e.id, label: e.name + " (" + team.name + ")" });
    }
  }
  const storyOpts = (storyList.stories || []).map((s) => ({
    id: s.id,
    label: s.jira_key ? s.title + " · " + s.jira_key : s.title,
  }));

  mountMultiChips(teamsEl, t.related_teams || [], teamOpts, async (next) => {
    const updated = await patchTask(t.id, { related_teams: next });
    Object.assign(t, updated);
    renderRelations(t);
  });
  mountMultiChips(peopleEl, t.related_employees || [], peopleOpts, async (next) => {
    const updated = await patchTask(t.id, { related_employees: next });
    Object.assign(t, updated);
    renderRelations(t);
  });
  mountStoryChips(storyEl, t.story_id || "", storyOpts, async (next) => {
    const updated = await patchTask(t.id, { story_id: next });
    Object.assign(t, updated);
    renderRelations(t);
  });
}

function chipHTML(id, label) {
  return (
    `<span class="chip" data-id="${escapeHtml(id)}">` +
    `<span class="chip-text">${escapeHtml(label)}</span>` +
    `<button type="button" class="chip-x" data-remove="${escapeHtml(id)}" aria-label="Remove">×</button>` +
    `</span>`
  );
}

function mountMultiChips(el, selected, options, onChange) {
  const selectedSet = new Set(selected);
  const labelFor = new Map(options.map((o) => [o.id, o.label]));
  const chips = selected.map((id) => chipHTML(id, labelFor.get(id) || id)).join("");
  const available = options.filter((o) => !selectedSet.has(o.id));
  el.innerHTML =
    chips +
    (available.length
      ? `<button type="button" class="chip-add" data-add aria-label="Add">+</button>`
      : "");

  el.querySelectorAll("[data-remove]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const id = btn.getAttribute("data-remove");
      const next = selected.filter((x) => x !== id);
      const done = memoryHold();
      try {
        await onChange(next);
        saveStatus.textContent = "";
      } catch (err) {
        saveStatus.textContent = err.message;
      } finally {
        done();
      }
    });
  });

  const addBtn = el.querySelector("[data-add]");
  if (addBtn) {
    addBtn.addEventListener("click", () => {
      openChipSelect(addBtn, available, async (id) => {
        const next = selected.concat([id]);
        const done = memoryHold();
        try {
          await onChange(next);
          saveStatus.textContent = "";
        } catch (err) {
          saveStatus.textContent = err.message;
        } finally {
          done();
        }
      });
    });
  }
}

function mountStoryChips(el, storyId, options, onChange) {
  const labelFor = new Map(options.map((o) => [o.id, o.label]));
  if (storyId) {
    el.innerHTML = chipHTML(storyId, labelFor.get(storyId) || storyId);
    el.querySelector("[data-remove]").addEventListener("click", async () => {
      const done = memoryHold();
      try {
        await onChange("");
        saveStatus.textContent = "";
      } catch (err) {
        saveStatus.textContent = err.message;
      } finally {
        done();
      }
    });
    return;
  }
  const available = options.slice();
  el.innerHTML = available.length
    ? `<button type="button" class="chip-add" data-add aria-label="Add story">+</button>`
    : `<span class="muted">—</span>`;
  const addBtn = el.querySelector("[data-add]");
  if (!addBtn) return;
  addBtn.addEventListener("click", () => {
    openChipSelect(addBtn, available, async (id) => {
      const done = memoryHold();
      try {
        await onChange(id);
        saveStatus.textContent = "";
      } catch (err) {
        saveStatus.textContent = err.message;
      } finally {
        done();
      }
    });
  });
}

function openChipSelect(addBtn, options, onPick) {
  if (addBtn.previousElementSibling?.classList?.contains("chip-select")) {
    addBtn.previousElementSibling.focus();
    return;
  }
  const select = document.createElement("select");
  select.className = "chip-select";
  select.setAttribute("aria-label", "Choose");
  const blank = document.createElement("option");
  blank.value = "";
  blank.textContent = "…";
  select.appendChild(blank);
  for (const o of options) {
    const opt = document.createElement("option");
    opt.value = o.id;
    opt.textContent = o.label;
    select.appendChild(opt);
  }
  addBtn.before(select);
  addBtn.hidden = true;
  select.focus();

  let closed = false;
  const close = () => {
    if (closed) return;
    closed = true;
    select.remove();
    addBtn.hidden = false;
  };

  select.addEventListener("change", async () => {
    const id = select.value;
    if (!id) {
      close();
      return;
    }
    select.disabled = true;
    try {
      await onPick(id);
    } catch (err) {
      saveStatus.textContent = err.message;
      select.disabled = false;
      return;
    }
    close();
  });
  select.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  });
  select.addEventListener("blur", () => {
    setTimeout(() => {
      if (!closed && document.activeElement !== select) close();
    }, 0);
  });
}

async function patchTask(id, body) {
  const res = await fetch("/api/tasks/" + encodeURIComponent(id), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

listMode = readListParam();
tabInbox.setAttribute("aria-selected", listMode === "inbox" ? "true" : "false");
tabArchive.setAttribute("aria-selected", listMode === "archive" ? "true" : "false");
newBtn.hidden = listMode !== "inbox";
if (listMode === "archive") {
  form.hidden = true;
  detailEl.hidden = false;
}

Promise.all([loadOrg(), loadStories(), refresh()]).catch((err) => {
  saveStatus.textContent = err.message;
});
