const listEl = document.getElementById("list");
const detailEl = document.getElementById("detail");
const form = document.getElementById("add-team");
const nameEl = document.getElementById("team-name");
const nicksEl = document.getElementById("team-nicks");
const saveBtn = document.getElementById("save-team");
const statusEl = document.getElementById("status");

const AUTOSAVE_MS = 650;
const pendingSaves = new Map();

let selectedId = null;
let org = { teams: [] };

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const name = nameEl.value.trim();
  if (!name) return;
  saveBtn.disabled = true;
  const done = memoryHold();
  try {
    const res = await fetch("/api/teams", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, nicknames: parseNicknames(nicksEl.value) }),
    });
    if (!res.ok) throw new Error(await res.text());
    const team = await res.json();
    nameEl.value = "";
    nicksEl.value = "";
    selectedId = team.id;
    statusEl.textContent = "";
    await refresh();
  } catch (err) {
    statusEl.textContent = "Save failed: " + err.message;
  } finally {
    done();
    saveBtn.disabled = false;
  }
});

async function refresh() {
  const res = await fetch("/api/org");
  if (!res.ok) throw new Error(await res.text());
  org = await res.json();
  if (!org.teams) org.teams = [];
  renderList();
  const team = org.teams.find((t) => t.id === selectedId);
  if (team) renderDetail(team);
  else {
    selectedId = null;
    detailEl.className = "detail empty";
    detailEl.innerHTML = `<p class="muted">Select a team.</p>`;
  }
}

function renderList() {
  listEl.innerHTML = "";
  if (!org.teams.length) {
    listEl.innerHTML = "<li class='muted' style='padding:0.8rem 0'>No teams yet.</li>";
    return;
  }
  for (const t of org.teams) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "item" + (t.id === selectedId ? " active" : "");
    const n = (t.employees || []).length;
    const nicks = nickLine(t.nicknames);
    btn.innerHTML =
      `<span class="item-title">${escapeHtml(t.name)}</span>` +
      (nicks ? `<span class="item-meta"><span>${escapeHtml(nicks)}</span></span>` : "") +
      `<span class="item-meta"><span>${n} ${n === 1 ? "person" : "people"}</span></span>`;
    btn.addEventListener("click", () => {
      selectedId = t.id;
      renderList();
      renderDetail(t);
    });
    li.appendChild(btn);
    listEl.appendChild(li);
  }
}

function renderDetail(t) {
  const people = t.employees || [];
  const items = people
    .map(
      (e) =>
        `<li>` +
        `<input type="text" data-emp-name="${escapeHtml(e.id)}" value="${escapeHtml(e.name)}" placeholder="name" autocomplete="off" />` +
        `<input type="text" data-emp-nicks="${escapeHtml(e.id)}" value="${escapeHtml((e.nicknames || []).join(", "))}" placeholder="nicknames" autocomplete="off" />` +
        `<button type="button" class="ghost" data-remove="${escapeHtml(e.id)}">Remove</button>` +
        `</li>`
    )
    .join("");
  detailEl.className = "detail";
  detailEl.innerHTML = `
    <div class="add-inline names">
      <div>
        <label for="edit-team-name">Team name</label>
        <input id="edit-team-name" type="text" value="${escapeHtml(t.name)}" required autocomplete="off" />
      </div>
      <div>
        <label for="edit-team-nicks">Team nicknames</label>
        <input id="edit-team-nicks" type="text" value="${escapeHtml((t.nicknames || []).join(", "))}" placeholder="plat, infra" autocomplete="off" />
      </div>
    </div>
    <ul class="people">${items || `<li class="muted">No people yet.</li>`}</ul>
    <form id="add-person" class="add-inline">
      <div>
        <label for="person-name">New person</label>
        <input id="person-name" type="text" required placeholder="Sara" autocomplete="off" />
      </div>
      <div>
        <label for="person-nicks">Nicknames</label>
        <input id="person-nicks" type="text" placeholder="sari, s" autocomplete="off" />
      </div>
      <button type="submit">Add</button>
    </form>
    <button type="button" class="ghost" id="remove-team">Remove team</button>
  `;
  bindAutosave(document.getElementById("edit-team-name"), "team-name:" + t.id, async () => {
    const name = document.getElementById("edit-team-name").value.trim();
    if (!name) throw new Error("name is empty");
    await patchJSON("/api/teams/" + encodeURIComponent(t.id), { name });
    const team = org.teams.find((x) => x.id === t.id);
    if (team) team.name = name;
    renderList();
  });
  bindAutosave(document.getElementById("edit-team-nicks"), "team-nicks:" + t.id, async () => {
    const nicknames = parseNicknames(document.getElementById("edit-team-nicks").value);
    await patchJSON("/api/teams/" + encodeURIComponent(t.id), { nicknames });
    const team = org.teams.find((x) => x.id === t.id);
    if (team) team.nicknames = nicknames;
    renderList();
  });
  detailEl.querySelectorAll("[data-emp-name]").forEach((input) => {
    const empId = input.getAttribute("data-emp-name");
    bindAutosave(input, "emp-name:" + t.id + ":" + empId, async () => {
      const name = input.value.trim();
      if (!name) throw new Error("name is empty");
      await patchJSON("/api/teams/" + encodeURIComponent(t.id) + "/employees/" + encodeURIComponent(empId), { name });
      const emp = findEmployee(t.id, empId);
      if (emp) emp.name = name;
      renderList();
    });
  });
  detailEl.querySelectorAll("[data-emp-nicks]").forEach((input) => {
    const empId = input.getAttribute("data-emp-nicks");
    bindAutosave(input, "emp-nicks:" + t.id + ":" + empId, async () => {
      const nicknames = parseNicknames(input.value);
      await patchJSON("/api/teams/" + encodeURIComponent(t.id) + "/employees/" + encodeURIComponent(empId), { nicknames });
      const emp = findEmployee(t.id, empId);
      if (emp) emp.nicknames = nicknames;
    });
  });
  const personForm = document.getElementById("add-person");
  const personName = document.getElementById("person-name");
  const personNicks = document.getElementById("person-nicks");
  personForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const name = personName.value.trim();
    if (!name) return;
    const done = memoryHold();
    try {
      const res = await fetch("/api/teams/" + encodeURIComponent(t.id) + "/employees", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, nicknames: parseNicknames(personNicks.value) }),
      });
      if (!res.ok) {
        statusEl.textContent = "Save failed: " + (await res.text());
        return;
      }
      statusEl.textContent = "";
      await refresh();
    } finally {
      done();
    }
  });
  detailEl.querySelectorAll("[data-remove]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const done = memoryHold();
      try {
        await fetch(
          "/api/teams/" + encodeURIComponent(t.id) + "/employees/" + encodeURIComponent(btn.getAttribute("data-remove")),
          { method: "DELETE" }
        );
        statusEl.textContent = "";
        await refresh();
      } finally {
        done();
      }
    });
  });
  document.getElementById("remove-team").addEventListener("click", async () => {
    const done = memoryHold();
    try {
      await fetch("/api/teams/" + encodeURIComponent(t.id), { method: "DELETE" });
      selectedId = null;
      statusEl.textContent = "";
      await refresh();
    } finally {
      done();
    }
  });
}

function findEmployee(teamId, empId) {
  const team = org.teams.find((x) => x.id === teamId);
  return (team?.employees || []).find((e) => e.id === empId);
}

async function patchJSON(url, body) {
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
}

function bindAutosave(input, key, persist) {
  const run = async () => {
    try {
      await persist();
      statusEl.textContent = "";
    } catch (err) {
      statusEl.textContent = "Save failed: " + err.message;
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

function parseNicknames(raw) {
  return String(raw || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

function nickLine(list) {
  return (list || []).join(", ");
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

refresh().catch((err) => {
  statusEl.textContent = "Could not load teams: " + err.message;
});
