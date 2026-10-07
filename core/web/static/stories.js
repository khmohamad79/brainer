const listEl = document.getElementById("list");
const detailEl = document.getElementById("detail");
const form = document.getElementById("add-story");
const titleEl = document.getElementById("story-title");
const saveBtn = document.getElementById("save-story");
const statusEl = document.getElementById("status");
const newBtn = document.getElementById("new-story");

const AUTOSAVE_MS = 650;
const pendingSaves = new Map();

let selectedId = null;
let stories = [];

function showComposer() {
  selectedId = null;
  form.hidden = false;
  detailEl.hidden = true;
  detailEl.innerHTML = "";
  renderList();
  titleEl.focus();
}

function showDetail() {
  form.hidden = true;
  detailEl.hidden = false;
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const title = titleEl.value.trim();
  if (!title) return;
  saveBtn.disabled = true;
  const done = memoryHold();
  try {
    const res = await fetch("/api/stories", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title }),
    });
    if (!res.ok) throw new Error(await res.text());
    const story = await res.json();
    titleEl.value = "";
    selectedId = story.id;
    statusEl.textContent = "";
    await refresh();
  } catch (err) {
    statusEl.textContent = err.message;
  } finally {
    done();
    saveBtn.disabled = false;
  }
});

newBtn.addEventListener("click", showComposer);

async function refresh() {
  const res = await fetch("/api/stories");
  if (!res.ok) throw new Error(await res.text());
  const data = await res.json();
  stories = data.stories || [];
  renderList();
  const story = stories.find((s) => s.id === selectedId);
  if (story) renderDetail(story);
  else if (selectedId) showComposer();
}

function renderList() {
  listEl.innerHTML = "";
  for (const s of stories) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "item" + (s.id === selectedId ? " active" : "");
    const meta = s.jira_key || "—";
    btn.innerHTML =
      `<span class="item-title">${escapeHtml(s.title)}</span>` +
      `<span class="item-meta"><span>${escapeHtml(meta)}</span></span>`;
    btn.addEventListener("click", () => {
      selectedId = s.id;
      renderList();
      renderDetail(s);
    });
    li.appendChild(btn);
    listEl.appendChild(li);
  }
}

function renderDetail(s) {
  showDetail();
  detailEl.innerHTML = `
    <div class="team-head">
      <input id="edit-story-title" class="team-name" type="text" value="${escapeHtml(s.title)}" required autocomplete="off" aria-label="Story title" />
      <button type="button" class="chip-x" id="remove-story" aria-label="Remove story">×</button>
    </div>
    <p class="section-label">Jira key</p>
    <input id="edit-jira-key" class="story-field" type="text" value="${escapeHtml(s.jira_key || "")}" autocomplete="off" aria-label="Jira key" />
    <p class="section-label">Summary</p>
    <textarea id="edit-summary" class="story-summary" rows="10" dir="auto" aria-label="Summary">${escapeHtml(s.summary || "")}</textarea>
    <div class="actions">
      <button type="button" class="ghost" id="jira-refresh"${s.jira_key ? "" : " disabled"}>Refresh from Jira</button>
    </div>
  `;

  bindAutosave(document.getElementById("edit-story-title"), "story-title:" + s.id, async () => {
    const title = document.getElementById("edit-story-title").value.trim();
    if (!title) throw new Error("title is empty");
    await patchJSON("/api/stories/" + encodeURIComponent(s.id), { title });
    s.title = title;
    renderList();
  });

  bindAutosave(document.getElementById("edit-jira-key"), "story-jira:" + s.id, async () => {
    const jira_key = document.getElementById("edit-jira-key").value.trim();
    const updated = await patchJSON("/api/stories/" + encodeURIComponent(s.id), { jira_key });
    Object.assign(s, updated);
    statusEl.textContent = "";
    renderDetail(s);
    renderList();
  });

  bindAutosave(document.getElementById("edit-summary"), "story-summary:" + s.id, async () => {
    const summary = document.getElementById("edit-summary").value;
    await patchJSON("/api/stories/" + encodeURIComponent(s.id), { summary });
    s.summary = summary;
  });

  document.getElementById("jira-refresh").addEventListener("click", async () => {
    const btn = document.getElementById("jira-refresh");
    btn.disabled = true;
    const done = memoryHold();
    try {
      const res = await fetch("/api/stories/" + encodeURIComponent(s.id) + "/jira-refresh", {
        method: "POST",
      });
      if (!res.ok) throw new Error(await res.text());
      const updated = await res.json();
      Object.assign(s, updated);
      statusEl.textContent = "";
      renderDetail(s);
      renderList();
    } catch (err) {
      statusEl.textContent = err.message;
      btn.disabled = false;
    } finally {
      done();
    }
  });

  document.getElementById("remove-story").addEventListener("click", async () => {
    const done = memoryHold();
    try {
      await fetch("/api/stories/" + encodeURIComponent(s.id), { method: "DELETE" });
      statusEl.textContent = "";
      showComposer();
      await refresh();
    } finally {
      done();
    }
  });
}

async function patchJSON(url, body) {
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

function bindAutosave(input, key, persist) {
  const run = async () => {
    try {
      await persist();
      statusEl.textContent = "";
    } catch (err) {
      statusEl.textContent = err.message;
    }
  };
  input.addEventListener("input", () => scheduleSave(key, run));
  input.addEventListener("blur", () => flushSave(key, run));
}

function scheduleSave(key, persist) {
  let entry = pendingSaves.get(key);
  if (!entry) {
    entry = { timer: 0, release: memoryHold() };
    pendingSaves.set(key, entry);
  }
  clearTimeout(entry.timer);
  entry.timer = setTimeout(async () => {
    pendingSaves.delete(key);
    try {
      await persist();
    } finally {
      entry.release();
    }
  }, AUTOSAVE_MS);
}

function flushSave(key, persist) {
  const entry = pendingSaves.get(key);
  if (!entry) return;
  clearTimeout(entry.timer);
  pendingSaves.delete(key);
  Promise.resolve(persist()).finally(entry.release);
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

refresh().catch((err) => {
  statusEl.textContent = err.message;
});
