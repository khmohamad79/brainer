const listEl = document.getElementById("list");
const detailEl = document.getElementById("detail");
const form = document.getElementById("add-team");
const nameEl = document.getElementById("team-name");
const saveBtn = document.getElementById("save-team");
const statusEl = document.getElementById("status");
const newBtn = document.getElementById("new-team");

const AUTOSAVE_MS = 650;
const pendingSaves = new Map();

let selectedId = null;
let org = { teams: [] };

function showComposer() {
  selectedId = null;
  form.hidden = false;
  detailEl.hidden = true;
  detailEl.innerHTML = "";
  renderList();
  nameEl.focus();
}

function showDetail() {
  form.hidden = true;
  detailEl.hidden = false;
}

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
      body: JSON.stringify({ name }),
    });
    if (!res.ok) throw new Error(await res.text());
    const team = await res.json();
    nameEl.value = "";
    selectedId = team.id;
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
  const res = await fetch("/api/org");
  if (!res.ok) throw new Error(await res.text());
  org = await res.json();
  if (!org.teams) org.teams = [];
  renderList();
  const team = org.teams.find((t) => t.id === selectedId);
  if (team) renderDetail(team);
  else if (selectedId) showComposer();
}

function renderList() {
  listEl.innerHTML = "";
  for (const t of org.teams) {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "item" + (t.id === selectedId ? " active" : "");
    const n = (t.employees || []).length;
    btn.innerHTML =
      `<span class="item-title">${escapeHtml(t.name)}</span>` +
      `<span class="item-meta"><span>${n}</span></span>`;
    btn.addEventListener("click", () => {
      selectedId = t.id;
      renderList();
      renderDetail(t);
    });
    li.appendChild(btn);
    listEl.appendChild(li);
  }
}

function nickChips(list, removeAttr) {
  return (list || [])
    .map(
      (n) =>
        `<span class="chip">` +
        `<span class="chip-text">${escapeHtml(n)}</span>` +
        `<button type="button" class="chip-x" ${removeAttr}="${escapeHtml(n)}" aria-label="Remove">×</button>` +
        `</span>`
    )
    .join("");
}

function renderDetail(t) {
  const people = t.employees || [];
  const items = people
    .map(
      (e) =>
        `<li data-emp="${escapeHtml(e.id)}">` +
        `<div class="person-head">` +
        `<input class="person-name" type="text" data-emp-name="${escapeHtml(e.id)}" value="${escapeHtml(e.name)}" autocomplete="off" aria-label="Name" />` +
        `<button type="button" class="chip-x person-remove" data-remove="${escapeHtml(e.id)}" aria-label="Remove person">×</button>` +
        `</div>` +
        `<div class="chip-row" data-emp-nicks="${escapeHtml(e.id)}">` +
        nickChips(e.nicknames, "data-remove-emp-nick") +
        `<button type="button" class="chip-add" data-add-emp-nick="${escapeHtml(e.id)}" aria-label="Add nickname">+</button>` +
        `</div>` +
        `</li>`
    )
    .join("");

  showDetail();
  detailEl.innerHTML = `
    <div class="team-head">
      <input id="edit-team-name" class="team-name" type="text" value="${escapeHtml(t.name)}" required autocomplete="off" aria-label="Team name" />
      <button type="button" class="chip-x" id="remove-team" aria-label="Remove team">×</button>
    </div>
    <p class="section-label">Nicknames</p>
    <div class="chip-row" id="team-nicks">
      ${nickChips(t.nicknames, "data-remove-team-nick")}
      <button type="button" class="chip-add" id="add-team-nick" aria-label="Add nickname">+</button>
    </div>
    <p class="section-label">People</p>
    <ul class="people">${items || `<li class="muted">—</li>`}</ul>
    <div class="chip-row">
      <button type="button" class="chip-add" id="add-person" aria-label="Add person">+</button>
    </div>
  `;

  bindAutosave(document.getElementById("edit-team-name"), "team-name:" + t.id, async () => {
    const name = document.getElementById("edit-team-name").value.trim();
    if (!name) throw new Error("name is empty");
    await patchJSON("/api/teams/" + encodeURIComponent(t.id), { name });
    const team = org.teams.find((x) => x.id === t.id);
    if (team) team.name = name;
    renderList();
  });

  document.getElementById("add-team-nick").addEventListener("click", (e) => {
    startChipInput(e.currentTarget, async (value) => {
      const next = uniqueAppend(t.nicknames || [], value);
      await patchJSON("/api/teams/" + encodeURIComponent(t.id), { nicknames: next });
      t.nicknames = next;
      statusEl.textContent = "";
      renderDetail(t);
      renderList();
    });
  });

  detailEl.querySelectorAll("[data-remove-team-nick]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const nick = btn.getAttribute("data-remove-team-nick");
      const next = (t.nicknames || []).filter((n) => n !== nick);
      const done = memoryHold();
      try {
        await patchJSON("/api/teams/" + encodeURIComponent(t.id), { nicknames: next });
        t.nicknames = next;
        statusEl.textContent = "";
        renderDetail(t);
        renderList();
      } catch (err) {
        statusEl.textContent = err.message;
      } finally {
        done();
      }
    });
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

  detailEl.querySelectorAll("[data-add-emp-nick]").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      const empId = btn.getAttribute("data-add-emp-nick");
      startChipInput(e.currentTarget, async (value) => {
        const emp = findEmployee(t.id, empId);
        if (!emp) return;
        const next = uniqueAppend(emp.nicknames || [], value);
        await patchJSON(
          "/api/teams/" + encodeURIComponent(t.id) + "/employees/" + encodeURIComponent(empId),
          { nicknames: next }
        );
        emp.nicknames = next;
        statusEl.textContent = "";
        renderDetail(t);
      });
    });
  });

  detailEl.querySelectorAll("[data-remove-emp-nick]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const li = btn.closest("[data-emp]");
      const empId = li?.getAttribute("data-emp");
      const nick = btn.getAttribute("data-remove-emp-nick");
      const emp = findEmployee(t.id, empId);
      if (!emp) return;
      const next = (emp.nicknames || []).filter((n) => n !== nick);
      const done = memoryHold();
      try {
        await patchJSON(
          "/api/teams/" + encodeURIComponent(t.id) + "/employees/" + encodeURIComponent(empId),
          { nicknames: next }
        );
        emp.nicknames = next;
        statusEl.textContent = "";
        renderDetail(t);
      } catch (err) {
        statusEl.textContent = err.message;
      } finally {
        done();
      }
    });
  });

  document.getElementById("add-person").addEventListener("click", (e) => {
    startChipInput(e.currentTarget, async (value) => {
      const res = await fetch("/api/teams/" + encodeURIComponent(t.id) + "/employees", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: value }),
      });
      if (!res.ok) throw new Error(await res.text());
      statusEl.textContent = "";
      await refresh();
    });
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
      statusEl.textContent = "";
      showComposer();
      await refresh();
    } finally {
      done();
    }
  });
}

function startChipInput(addBtn, onCommit) {
  if (addBtn.previousElementSibling?.classList?.contains("chip-input")) {
    addBtn.previousElementSibling.focus();
    return;
  }
  const input = document.createElement("input");
  input.type = "text";
  input.className = "chip-input";
  input.autocomplete = "off";
  addBtn.before(input);
  input.focus();

  let closed = false;
  const close = () => {
    if (closed) return;
    closed = true;
    input.remove();
  };

  const commit = async () => {
    const value = input.value.trim();
    if (!value) {
      close();
      return;
    }
    input.disabled = true;
    const done = memoryHold();
    try {
      await onCommit(value);
    } catch (err) {
      statusEl.textContent = err.message;
      input.disabled = false;
      input.focus();
      return;
    } finally {
      done();
    }
    close();
  };

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      commit();
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  });
  input.addEventListener("blur", () => {
    setTimeout(() => {
      if (!closed && document.activeElement !== input) commit();
    }, 0);
  });
}

function uniqueAppend(list, value) {
  const next = list.slice();
  if (!next.some((n) => n.toLowerCase() === value.toLowerCase())) next.push(value);
  return next;
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
