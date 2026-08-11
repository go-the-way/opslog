(() => {
  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  const TABS = ["logs", "tail", "metrics", "probes", "configs", "archives"];
  const SIGNAL_STORE_PREFIX = "opslog:signal:";
  const SIGNAL_RETURN_KEY = "opslog:signal-return";
  const TAIL_STATE_KEY = "opslog:tail-state";
  const DIAG_ATTR_KEYS = [
    "profile",
    "version",
    "git_sha",
    "hostname",
    "pid",
    "cwd",
    "go_version",
    "goos",
    "goarch",
  ];
  const SPECIAL_ATTR_KEYS = new Set([
    "stack",
    "panic",
    "env",
    "environ",
    "startup_environ",
    "system_environ",
    "process_environ",
    "sys",
    ...DIAG_ATTR_KEYS,
  ]);

  const state = {
    logOffset: 0,
    logLimit: 50,
    logHasMore: false,
    loaded: {},
    tailWS: null,
    tailCount: 0,
    tailStopping: false, // intentional Stop / reconnect swap
    pageUnloading: false,
    activeRange: "",
    view: "tab", // "tab" | "signal"
    returnTab: "logs",
    currentSignalId: "",
  };

  function escapeHtml(s) {
    return String(s ?? "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function fmtTime(ts) {
    if (!ts) return "—";
    const d = new Date(ts);
    if (Number.isNaN(d.getTime())) return String(ts);
    return d.toLocaleString(undefined, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    });
  }

  function fmtBytes(n) {
    const v = Number(n) || 0;
    if (v < 1024) return `${v} B`;
    if (v < 1024 ** 2) return `${(v / 1024).toFixed(1)} KB`;
    if (v < 1024 ** 3) return `${(v / 1024 ** 2).toFixed(1)} MB`;
    return `${(v / 1024 ** 3).toFixed(2)} GB`;
  }

  function levelClass(level) {
    const l = String(level || "").toLowerCase();
    if (["error", "fatal", "panic"].includes(l)) return "error";
    if (["warn", "warning"].includes(l)) return "warn";
    if (l === "info") return "info";
    if (["debug", "trace"].includes(l)) return "debug";
    if (["ok", "success", "up", "pass"].includes(l)) return "ok";
    return "debug";
  }

  function toLocalInputValue(d) {
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }

  function localToRFC3339(localVal) {
    if (!localVal) return "";
    const d = new Date(localVal);
    if (Number.isNaN(d.getTime())) return "";
    return d.toISOString();
  }

  function formParams(form, extras = {}) {
    const fd = new FormData(form);
    const p = new URLSearchParams();
    for (const [k, v] of fd.entries()) {
      let val = String(v).trim();
      if (!val) continue;
      if (k === "from" || k === "to") val = localToRFC3339(val);
      if (val) p.set(k, val);
    }
    for (const [k, v] of Object.entries(extras)) {
      if (v != null && v !== "") p.set(k, String(v));
    }
    return p;
  }

  function setWrapState(wrapId, mode, errorMsg) {
    const wrap = $(wrapId);
    if (!wrap) return;
    wrap.dataset.state = mode;
    const prefix = wrapId.replace("-wrap", "");
    const empty = $(`${prefix}-empty`);
    const err = $(`${prefix}-error`);
    const loading = $(`${prefix}-loading`);
    if (empty) empty.classList.toggle("hidden", mode !== "empty");
    if (err) {
      err.classList.toggle("hidden", mode !== "error");
      if (mode === "error" && errorMsg != null) err.textContent = errorMsg;
    }
    if (loading) loading.classList.toggle("hidden", mode !== "loading");
  }

  function formatAttrValue(val) {
    if (val == null) return "—";
    if (typeof val === "string") return val;
    try {
      return JSON.stringify(val, null, 2);
    } catch {
      return String(val);
    }
  }

  function newSignalId() {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
      return crypto.randomUUID();
    }
    return `s-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
  }

  function storeSignal(obj) {
    const id = newSignalId();
    try {
      sessionStorage.setItem(SIGNAL_STORE_PREFIX + id, JSON.stringify(obj));
    } catch {
      /* quota / private mode — detail may show missing */
    }
    return id;
  }

  function loadStoredSignal(id) {
    if (!id) return null;
    try {
      const raw = sessionStorage.getItem(SIGNAL_STORE_PREFIX + id);
      if (!raw) return null;
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  function parseHash() {
    const raw = location.hash.replace(/^#/, "");
    if (!raw) return { type: "tab", tab: "logs" };
    const signalMatch = raw.match(/^signal\/([^/?#]+)$/);
    if (signalMatch) return { type: "signal", id: decodeURIComponent(signalMatch[1]) };
    if (TABS.includes(raw)) return { type: "tab", tab: raw };
    return { type: "tab", tab: "logs" };
  }

  function openRawModal(obj) {
    const dlg = $("#detail-dialog");
    $("#detail-body").textContent = JSON.stringify(obj, null, 2);
    if (typeof dlg.showModal === "function") dlg.showModal();
  }

  function fillKvTable(tbody, rows) {
    tbody.innerHTML = "";
    for (const [key, value] of rows) {
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td class="kv-key">${escapeHtml(key)}</td>
        <td class="kv-val">${escapeHtml(value)}</td>
      `;
      tbody.appendChild(tr);
    }
  }

  const SIGNAL_TABS = ["overview", "runtime", "env", "stack", "raw"];
  const ENV_TABS = ["startup", "system", "process"];

  function switchSignalTab(name) {
    if (!SIGNAL_TABS.includes(name)) name = "overview";
    $$(".signal-tab").forEach((btn) => {
      const on = btn.dataset.signalTab === name;
      btn.classList.toggle("active", on);
      btn.setAttribute("aria-selected", on ? "true" : "false");
    });
    $$(".signal-tab-panel").forEach((panel) => {
      const on = panel.dataset.signalPanel === name;
      panel.classList.toggle("active", on);
      panel.hidden = !on;
    });
  }

  function switchEnvTab(name) {
    if (!ENV_TABS.includes(name)) name = "startup";
    $$(".signal-subtab").forEach((btn) => {
      const on = btn.dataset.envTab === name;
      btn.classList.toggle("active", on);
      btn.setAttribute("aria-selected", on ? "true" : "false");
    });
    $$(".signal-env-panel").forEach((panel) => {
      const on = panel.dataset.envPanel === name;
      panel.classList.toggle("active", on);
      panel.hidden = !on;
    });
  }

  function fillEnvSection(sectionSel, tableSel, emptySel, source) {
    const empty = $(emptySel);
    const table = $(tableSel);
    if (!table || !empty) return;
    if (!source || !Object.keys(source).length) {
      table.classList.add("hidden");
      empty.classList.remove("hidden");
      $(`${tableSel} tbody`).innerHTML = "";
      return;
    }
    empty.classList.add("hidden");
    table.classList.remove("hidden");
    const rows = Object.keys(source)
      .sort((a, b) => a.localeCompare(b))
      .map((k) => [k, formatAttrValue(source[k])]);
    fillKvTable($(`${tableSel} tbody`), rows);
  }

  function renderEnvironment(attrs) {
    const startup =
      attrs.startup_environ && typeof attrs.startup_environ === "object"
        ? attrs.startup_environ
        : null;
    const system =
      attrs.system_environ && typeof attrs.system_environ === "object"
        ? attrs.system_environ
        : null;
    const process =
      attrs.process_environ && typeof attrs.process_environ === "object"
        ? attrs.process_environ
        : attrs.environ && typeof attrs.environ === "object"
          ? attrs.environ
          : attrs.env && typeof attrs.env === "object"
            ? attrs.env
            : null;

    fillEnvSection(
      "#signal-startup-env-section",
      "#signal-startup-env",
      "#signal-startup-env-empty",
      startup
    );
    fillEnvSection(
      "#signal-system-env-section",
      "#signal-system-env",
      "#signal-system-env-empty",
      system
    );
    fillEnvSection(
      "#signal-process-env-section",
      "#signal-process-env",
      "#signal-process-env-empty",
      process
    );
  }

  function renderSystemStatus(sys) {
    const section = $("#signal-sys-section");
    if (!sys || typeof sys !== "object") {
      section.classList.add("hidden");
      return;
    }
    section.classList.remove("hidden");

    const skip = new Set(["disk", "net", "net_addrs"]);
    const prefer = [
      "pid",
      "ppid",
      "goroutines",
      "num_cpu",
      "proc_rss_human",
      "proc_vms_human",
      "proc_fds",
      "mem_alloc_human",
      "mem_sys_human",
      "mem_heap_inuse",
      "gc_num",
      "cwd",
      "go_version",
      "goos",
      "goarch",
    ];
    const rows = [];
    for (const k of prefer) {
      if (sys[k] != null && sys[k] !== "") rows.push([k, formatAttrValue(sys[k])]);
    }
    for (const k of Object.keys(sys).sort()) {
      if (skip.has(k) || prefer.includes(k)) continue;
      if (typeof sys[k] === "object") continue;
      rows.push([k, formatAttrValue(sys[k])]);
    }
    fillKvTable($("#signal-sys tbody"), rows);

    const disk = sys.disk;
    const diskWrap = $("#signal-disk-wrap");
    if (disk && typeof disk === "object") {
      diskWrap.classList.remove("hidden");
      fillKvTable(
        $("#signal-disk tbody"),
        Object.keys(disk)
          .sort()
          .map((k) => [k, formatAttrValue(disk[k])])
      );
    } else {
      diskWrap.classList.add("hidden");
      $("#signal-disk tbody").innerHTML = "";
    }

    const nets = Array.isArray(sys.net) ? sys.net : [];
    const netWrap = $("#signal-net-wrap");
    const netBody = $("#signal-net tbody");
    if (nets.length) {
      netWrap.classList.remove("hidden");
      netBody.innerHTML = nets
        .map((n) => {
          return `<tr>
            <td class="mono">${escapeHtml(n.name || "—")}</td>
            <td class="mono">${escapeHtml(n.rx_human || fmtBytes(n.rx_bytes))}</td>
            <td class="mono">${escapeHtml(n.tx_human || fmtBytes(n.tx_bytes))}</td>
            <td class="mono">${escapeHtml(n.rx_packets ?? "—")}</td>
            <td class="mono">${escapeHtml(n.tx_packets ?? "—")}</td>
          </tr>`;
        })
        .join("");
    } else {
      netWrap.classList.add("hidden");
      netBody.innerHTML = "";
    }

    const addrs = Array.isArray(sys.net_addrs) ? sys.net_addrs : [];
    const addrWrap = $("#signal-addrs-wrap");
    if (addrs.length) {
      addrWrap.classList.remove("hidden");
      fillKvTable(
        $("#signal-addrs tbody"),
        addrs.map((a) => [
          a.name || "iface",
          Array.isArray(a.addrs) ? a.addrs.join(", ") : formatAttrValue(a),
        ])
      );
    } else {
      addrWrap.classList.add("hidden");
      $("#signal-addrs tbody").innerHTML = "";
    }
  }

  function renderSignalDetail(obj) {
    const detail = $("#signal-detail");
    const missing = $("#signal-missing");
    if (!obj) {
      detail.classList.add("hidden");
      missing.classList.remove("hidden");
      return;
    }
    missing.classList.add("hidden");
    detail.classList.remove("hidden");
    switchSignalTab("overview");
    switchEnvTab("startup");

    const kind = obj.kind || "log";
    const level = obj.level || "—";
    const lvl = levelClass(level);
    $("#signal-kind").textContent = kind;
    const levelEl = $("#signal-level");
    levelEl.textContent = level;
    levelEl.className = `level-badge ${lvl}`;
    $("#signal-time").textContent = fmtTime(obj.ts);
    $("#signal-msg").textContent = obj.msg || "(no message)";
    $("#signal-service").textContent = obj.service || "—";
    $("#signal-host").textContent = obj.host || "—";
    $("#signal-trace").textContent = obj.trace_id || "—";

    const fieldRows = [
      ["kind", String(kind)],
      ["level", String(level)],
      ["ts", obj.ts ? String(obj.ts) : "—"],
      ["service", obj.service || "—"],
      ["host", obj.host || "—"],
      ["trace_id", obj.trace_id || "—"],
      ["msg", obj.msg || "—"],
    ];
    fillKvTable($("#signal-fields tbody"), fieldRows);

    const attrs = obj.attrs && typeof obj.attrs === "object" ? obj.attrs : {};
    const stackVal = attrs.stack;
    const panicVal = attrs.panic;

    const diagRows = DIAG_ATTR_KEYS
      .filter((k) => attrs[k] != null && String(attrs[k]) !== "")
      .map((k) => [k, formatAttrValue(attrs[k])]);
    const diagSection = $("#signal-diag-section");
    if (diagRows.length) {
      diagSection.classList.remove("hidden");
      fillKvTable($("#signal-diag tbody"), diagRows);
    } else {
      diagSection.classList.add("hidden");
      $("#signal-diag tbody").innerHTML = "";
    }

    renderSystemStatus(attrs.sys);
    const runtimeEmpty = $("#signal-runtime-empty");
    const hasRuntime =
      diagRows.length ||
      (attrs.sys && typeof attrs.sys === "object");
    runtimeEmpty.classList.toggle("hidden", !!hasRuntime);

    renderEnvironment(attrs);

    const panicSection = $("#signal-panic-section");
    const hasPanic = panicVal != null && String(panicVal) !== "";
    if (hasPanic) {
      panicSection.classList.remove("hidden");
      $("#signal-panic").textContent = formatAttrValue(panicVal);
    } else {
      panicSection.classList.add("hidden");
      $("#signal-panic").textContent = "";
    }

    const stackSection = $("#signal-stack-section");
    const hasStack = stackVal != null && String(stackVal) !== "";
    if (hasStack) {
      stackSection.classList.remove("hidden");
      $("#signal-stack").textContent = formatAttrValue(stackVal);
    } else {
      stackSection.classList.add("hidden");
      $("#signal-stack").textContent = "";
    }
    $("#signal-stack-empty").classList.toggle("hidden", hasPanic || hasStack);

    const attrRows = Object.keys(attrs)
      .filter((k) => !SPECIAL_ATTR_KEYS.has(k))
      .sort((a, b) => a.localeCompare(b))
      .map((k) => [k, formatAttrValue(attrs[k])]);
    const attrsEmpty = $("#signal-attrs-empty");
    const attrsTable = $("#signal-attrs");
    if (!attrRows.length) {
      attrsTable.classList.add("hidden");
      attrsEmpty.classList.remove("hidden");
      $("#signal-attrs tbody").innerHTML = "";
    } else {
      attrsTable.classList.remove("hidden");
      attrsEmpty.classList.add("hidden");
      fillKvTable($("#signal-attrs tbody"), attrRows);
    }

    $("#signal-raw-body").textContent = JSON.stringify(obj, null, 2);
    detail._signal = obj;
  }

  function showSignalView(id) {
    state.view = "signal";
    state.currentSignalId = id;
    $$(".nav-btn").forEach((b) => b.classList.remove("active"));
    $$(".panel").forEach((p) => {
      const isSignal = p.dataset.panel === "signal";
      p.classList.toggle("active", isSignal);
      if (isSignal) p.hidden = false;
    });
    renderSignalDetail(loadStoredSignal(id));
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function hideSignalView() {
    const panel = $("#panel-signal");
    if (panel) {
      panel.classList.remove("active");
      panel.hidden = true;
    }
    state.view = "tab";
    state.currentSignalId = "";
  }

  function openSignalDetail(obj, opts = {}) {
    const id = storeSignal(obj);
    const fromTab =
      opts.fromTab ||
      (state.view === "tab"
        ? $$(".nav-btn.active")[0]?.dataset.tab || state.returnTab || "logs"
        : state.returnTab || "logs");
    state.returnTab = fromTab;
    try {
      sessionStorage.setItem(SIGNAL_RETURN_KEY, fromTab);
    } catch {
      /* ignore */
    }
    const next = `#signal/${encodeURIComponent(id)}`;
    if (location.hash !== next) {
      history.pushState({ signalId: id, returnTab: fromTab }, "", next);
    }
    showSignalView(id);
  }

  function closeSignalDetail() {
    let tab = state.returnTab;
    try {
      tab = sessionStorage.getItem(SIGNAL_RETURN_KEY) || tab;
    } catch {
      /* ignore */
    }
    if (!TABS.includes(tab)) tab = "logs";
    hideSignalView();
    const next = `#${tab}`;
    if (location.hash !== next) history.pushState(null, "", next);
    switchTab(tab, false);
  }

  async function fetchJSON(url, opts) {
    const res = await fetch(url, opts);
    const text = await res.text();
    let data = null;
    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      data = { raw: text };
    }
    if (!res.ok) {
      const msg = (data && (data.error || data.message || data.raw)) || text || res.statusText;
      const err = new Error(typeof msg === "string" ? msg : `HTTP ${res.status}`);
      err.status = res.status;
      throw err;
    }
    return data;
  }

  /* —— Navigation —— */
  function switchTab(tab, pushHash = true) {
    if (!TABS.includes(tab)) tab = "logs";
    hideSignalView();
    state.returnTab = tab;
    $$(".nav-btn").forEach((b) => b.classList.toggle("active", b.dataset.tab === tab));
    $$(".panel").forEach((p) => {
      const on = p.dataset.panel === tab;
      p.classList.toggle("active", on);
      if (p.dataset.panel === "signal") p.hidden = true;
    });
    if (pushHash) {
      const next = `#${tab}`;
      if (location.hash !== next) history.replaceState(null, "", next);
    }
    if (!state.loaded[tab]) {
      state.loaded[tab] = true;
      if (tab === "metrics") loadMetrics();
      else if (tab === "probes") loadProbes();
      else if (tab === "configs") loadConfigs();
      else if (tab === "archives") loadArchives();
      else if (tab === "logs") searchLogs(true);
    }
  }

  function applyRoute() {
    const route = parseHash();
    if (route.type === "signal") {
      try {
        const saved = sessionStorage.getItem(SIGNAL_RETURN_KEY);
        if (saved && TABS.includes(saved)) state.returnTab = saved;
      } catch {
        /* ignore */
      }
      showSignalView(route.id);
      return;
    }
    switchTab(route.tab, false);
  }

  $$(".nav-btn").forEach((btn) => {
    btn.addEventListener("click", () => switchTab(btn.dataset.tab));
  });

  window.addEventListener("hashchange", () => applyRoute());
  window.addEventListener("popstate", () => applyRoute());

  $("#signal-back").addEventListener("click", () => closeSignalDetail());
  $("#signal-raw-toggle").addEventListener("click", () => {
    switchSignalTab("raw");
    window.scrollTo({ top: 0, behavior: "smooth" });
  });
  $$(".signal-tab").forEach((btn) => {
    btn.addEventListener("click", () => switchSignalTab(btn.dataset.signalTab));
  });
  $$(".signal-subtab").forEach((btn) => {
    btn.addEventListener("click", () => switchEnvTab(btn.dataset.envTab));
  });

  /* —— Health + clock —— */
  async function pollHealth() {
    const badge = $("#health-badge");
    try {
      const data = await fetchJSON("/api/health");
      const ok = data && data.ok !== false;
      badge.dataset.state = ok ? "ok" : "down";
      $(".health-label", badge).textContent = ok ? "Healthy" : "Degraded";
    } catch {
      badge.dataset.state = "down";
      $(".health-label", badge).textContent = "Unreachable";
    }
  }

  function tickClock() {
    const el = $("#clock");
    if (!el) return;
    el.textContent = new Date().toLocaleString(undefined, {
      weekday: "short",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    });
  }

  /* —— Shared renderers —— */
  function attrsPreview(attrs) {
    if (!attrs || typeof attrs !== "object") return "—";
    const keys = Object.keys(attrs);
    if (!keys.length) return "—";
    const preview = keys
      .slice(0, 3)
      .map((k) => `${k}=${typeof attrs[k] === "object" ? "{…}" : attrs[k]}`)
      .join(", ");
    return escapeHtml(preview + (keys.length > 3 ? "…" : ""));
  }

  /** Compact identity for tooltips: service@host when both present. */
  function identityTitle(it) {
    const svc = it.service || "";
    const host = it.host || "";
    if (svc && host) return `${svc}@${host}`;
    return svc || host || "";
  }

  function bindDetailBtn(tr, it, fromTab) {
    $(".detail-btn", tr).addEventListener("click", () => {
      openSignalDetail(it, { fromTab });
    });
  }

  function renderLogRows(items, tbody) {
    tbody.innerHTML = "";
    for (const it of items) {
      const tr = document.createElement("tr");
      const lvl = levelClass(it.level);
      const svc = it.service || "—";
      const host = it.host || "—";
      const idTitle = identityTitle(it);
      tr.innerHTML = `
        <td class="mono cell-clip">${escapeHtml(fmtTime(it.ts))}</td>
        <td class="mono cell-level ${lvl}">${escapeHtml(it.level || "—")}</td>
        <td class="mono cell-clip" title="${escapeHtml(idTitle || svc)}">${escapeHtml(svc)}</td>
        <td class="mono cell-clip cell-host" title="${escapeHtml(host)}">${escapeHtml(host)}</td>
        <td class="msg">${escapeHtml(it.msg || "")}</td>
        <td class="cell-action"><button type="button" class="btn ghost detail-btn">Detail</button></td>
      `;
      bindDetailBtn(tr, it, "logs");
      tbody.appendChild(tr);
    }
  }

  function renderMetricRows(items, tbody) {
    tbody.innerHTML = "";
    for (const it of items) {
      const tr = document.createElement("tr");
      const svc = it.service || "—";
      const host = it.host || "—";
      const idTitle = identityTitle(it);
      tr.innerHTML = `
        <td class="mono cell-clip">${escapeHtml(fmtTime(it.ts))}</td>
        <td class="mono cell-clip" title="${escapeHtml(idTitle || svc)}">${escapeHtml(svc)}</td>
        <td class="mono cell-clip cell-host" title="${escapeHtml(host)}">${escapeHtml(host)}</td>
        <td class="msg">${escapeHtml(it.msg || "")}</td>
        <td class="mono cell-clip">${attrsPreview(it.attrs)}</td>
        <td class="cell-action"><button type="button" class="btn ghost detail-btn">Detail</button></td>
      `;
      bindDetailBtn(tr, it, "metrics");
      tbody.appendChild(tr);
    }
  }

  function renderProbeRows(items, tbody) {
    tbody.innerHTML = "";
    for (const it of items) {
      const tr = document.createElement("tr");
      const lvl = levelClass(it.level);
      const svc = it.service || "—";
      const host = it.host || "—";
      const idTitle = identityTitle(it);
      tr.innerHTML = `
        <td class="mono cell-clip">${escapeHtml(fmtTime(it.ts))}</td>
        <td class="mono cell-level ${lvl}">${escapeHtml(it.level || "—")}</td>
        <td class="mono cell-clip" title="${escapeHtml(idTitle || svc)}">${escapeHtml(svc)}</td>
        <td class="mono cell-clip cell-host" title="${escapeHtml(host)}">${escapeHtml(host)}</td>
        <td class="msg">${escapeHtml(it.msg || "")}</td>
        <td class="mono cell-clip">${attrsPreview(it.attrs)}</td>
        <td class="cell-action"><button type="button" class="btn ghost detail-btn">Detail</button></td>
      `;
      bindDetailBtn(tr, it, "probes");
      tbody.appendChild(tr);
    }
  }

  function renderConfigRows(items, tbody) {
    tbody.innerHTML = "";
    for (const it of items) {
      const tr = document.createElement("tr");
      const svc = it.service || "—";
      const host = it.host || "—";
      const idTitle = identityTitle(it);
      tr.innerHTML = `
        <td class="mono cell-clip">${escapeHtml(fmtTime(it.ts))}</td>
        <td class="mono cell-clip" title="${escapeHtml(idTitle || svc)}">${escapeHtml(svc)}</td>
        <td class="mono cell-clip cell-host" title="${escapeHtml(host)}">${escapeHtml(host)}</td>
        <td class="msg">${escapeHtml(it.msg || "config snapshot")}</td>
        <td class="cell-action"><button type="button" class="btn ghost detail-btn">View</button></td>
      `;
      bindDetailBtn(tr, it, "configs");
      tbody.appendChild(tr);
    }
  }

  async function loadSignalPanel(wrapId, path, tbodySel, renderer) {
    const tbody = $(tbodySel);
    setWrapState(wrapId, "loading");
    tbody.innerHTML = "";
    try {
      const data = await fetchJSON(path);
      const items = data.items || (Array.isArray(data) ? data : []);
      if (!items.length) {
        setWrapState(wrapId, "empty");
        return;
      }
      renderer(items, tbody);
      setWrapState(wrapId, "ready");
    } catch (e) {
      const msg =
        e.status === 503
          ? "Queryable output is not configured on this server."
          : e.message || "Failed to load";
      setWrapState(wrapId, "error", msg);
    }
  }

  function loadMetrics() {
    return loadSignalPanel("#metrics-wrap", "/api/metrics?limit=100", "#metrics-results", renderMetricRows);
  }

  function loadProbes() {
    return loadSignalPanel("#probes-wrap", "/api/probes?limit=100", "#probes-results", renderProbeRows);
  }

  function loadConfigs() {
    return loadSignalPanel("#configs-wrap", "/api/configs?limit=50", "#configs-results", renderConfigRows);
  }

  /* —— Logs —— */
  function applyRange(key) {
    const form = $("#log-form");
    const from = form.elements.from;
    const to = form.elements.to;
    $$("#log-ranges .chip").forEach((c) => c.classList.toggle("active", c.dataset.range === key && key !== "clear"));
    state.activeRange = key === "clear" ? "" : key;
    if (key === "clear") {
      from.value = "";
      to.value = "";
      return;
    }
    const now = new Date();
    const start = new Date(now);
    if (key === "1h") start.setHours(start.getHours() - 1);
    else if (key === "6h") start.setHours(start.getHours() - 6);
    else if (key === "24h") start.setHours(start.getHours() - 24);
    else if (key === "7d") start.setDate(start.getDate() - 7);
    from.value = toLocalInputValue(start);
    to.value = toLocalInputValue(now);
  }

  $$("#log-ranges .chip").forEach((btn) => {
    btn.addEventListener("click", () => {
      applyRange(btn.dataset.range);
      searchLogs(true);
    });
  });

  async function searchLogs(reset) {
    if (reset) state.logOffset = 0;
    const form = $("#log-form");
    const p = formParams(form, {
      kind: "log",
      limit: state.logLimit,
      offset: state.logOffset,
    });
    const tbody = $("#log-results");
    setWrapState("#log-wrap", "loading");
    tbody.innerHTML = "";
    try {
      const data = await fetchJSON(`/api/signals?${p}`);
      const items = data.items || [];
      state.logHasMore = !!data.has_more;
      if ($("#log-meta")) {
        $("#log-meta").textContent = `total ${data.total ?? items.length} · offset ${state.logOffset}${
          data.has_more ? " · more" : ""
        }`;
      }
      $("#log-page").textContent = `Page ${Math.floor(state.logOffset / state.logLimit) + 1}`;
      $("#log-prev").disabled = state.logOffset <= 0;
      $("#log-next").disabled = !state.logHasMore;
      if (!items.length) {
        setWrapState("#log-wrap", "empty");
        return;
      }
      renderLogRows(items, tbody);
      setWrapState("#log-wrap", "ready");
    } catch (e) {
      if ($("#log-meta")) $("#log-meta").textContent = "error";
      $("#log-prev").disabled = true;
      $("#log-next").disabled = true;
      const msg =
        e.status === 503
          ? "Queryable output is not configured on this server."
          : e.message || "Search failed";
      setWrapState("#log-wrap", "error", msg);
    }
  }

  $("#log-form").addEventListener("submit", (e) => {
    e.preventDefault();
    searchLogs(true);
  });

  $("#log-prev").addEventListener("click", () => {
    state.logOffset = Math.max(0, state.logOffset - state.logLimit);
    searchLogs(false);
  });

  $("#log-next").addEventListener("click", () => {
    if (!state.logHasMore) return;
    state.logOffset += state.logLimit;
    searchLogs(false);
  });

  /* —— Live Tail —— */
  function setTailStatus(stateName, label) {
    const el = $("#tail-status");
    el.dataset.state = stateName;
    $(".live-label", el).textContent = label;
  }

  function updateTailCount() {
    $("#tail-count").textContent = `${state.tailCount} event${state.tailCount === 1 ? "" : "s"}`;
  }

  function readTailState() {
    try {
      const raw = sessionStorage.getItem(TAIL_STATE_KEY);
      if (!raw) return {};
      const obj = JSON.parse(raw);
      return obj && typeof obj === "object" ? obj : {};
    } catch {
      return {};
    }
  }

  function writeTailState(patch) {
    try {
      const next = { ...readTailState(), ...patch, updatedAt: Date.now() };
      sessionStorage.setItem(TAIL_STATE_KEY, JSON.stringify(next));
    } catch {
      /* ignore */
    }
  }

  function collectTailFilters() {
    const form = $("#tail-form");
    const fd = new FormData(form);
    return {
      level: String(fd.get("level") || ""),
      service: String(fd.get("service") || ""),
      host: String(fd.get("host") || ""),
      keyword: String(fd.get("keyword") || ""),
      autoscroll: !!$("#tail-autoscroll").checked,
    };
  }

  function applyTailFilters(saved) {
    if (!saved || typeof saved !== "object") return;
    const form = $("#tail-form");
    if ("level" in saved) form.elements.level.value = saved.level || "";
    if ("service" in saved) form.elements.service.value = saved.service || "";
    if ("host" in saved) form.elements.host.value = saved.host || "";
    if ("keyword" in saved) form.elements.keyword.value = saved.keyword || "";
    if ("autoscroll" in saved) $("#tail-autoscroll").checked = !!saved.autoscroll;
  }

  function appendTailLine(it) {
    const box = $("#tail-results");
    $(".tail-placeholder", box)?.remove();
    const line = document.createElement("div");
    line.className = "tail-line";
    line.tabIndex = 0;
    line.setAttribute("role", "button");
    const lvl = levelClass(it.level);
    const svc = it.service || "—";
    const host = it.host || "—";
    const idTitle = identityTitle(it);
    line.innerHTML = `
      <span class="ts">${escapeHtml(fmtTime(it.ts))}</span>
      <span class="lvl ${lvl}">${escapeHtml(it.level || "—")}</span>
      <span class="svc" title="${escapeHtml(idTitle || svc)}">${escapeHtml(svc)}</span>
      <span class="host" title="${escapeHtml(host)}">${escapeHtml(host)}</span>
      <span class="msg">${escapeHtml(it.msg || "")}</span>
    `;
    const open = () => openSignalDetail(it, { fromTab: "tail" });
    line.addEventListener("click", open);
    line.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        open();
      }
    });
    box.prepend(line);
    while (box.children.length > 500) box.removeChild(box.lastChild);
    state.tailCount += 1;
    updateTailCount();
    if ($("#tail-autoscroll").checked) box.scrollTop = 0;
  }

  function stopTail(opts = {}) {
    const intentional = !!opts.intentional;
    const reconnecting = !!opts.reconnecting;
    state.tailStopping = intentional || reconnecting;
    if (state.tailWS) {
      const ws = state.tailWS;
      state.tailWS = null;
      try {
        ws.close();
      } catch {
        /* ignore */
      }
    }
    if (!reconnecting) {
      $("#tail-stop").disabled = true;
      $("#tail-connect").disabled = false;
    }
    if (intentional) {
      writeTailState({ ...collectTailFilters(), live: false });
    }
  }

  function startTail(opts = {}) {
    const restored = !!opts.restored;
    stopTail({ reconnecting: true });
    const form = $("#tail-form");
    const filters = collectTailFilters();
    writeTailState({ ...filters, live: true });
    const p = formParams(form);
    const proto = location.protocol === "https:" ? "wss" : "ws";
    const url = `${proto}://${location.host}/api/tail?${p}`;
    const box = $("#tail-results");
    if (!box.querySelector(".tail-line")) {
      box.innerHTML = `<div class="tail-placeholder">${
        restored ? "Reconnecting…" : "Connecting…"
      }</div>`;
    }
    setTailStatus("idle", restored ? "Reconnecting…" : "Connecting…");
    state.tailStopping = false;
    const ws = new WebSocket(url);
    state.tailWS = ws;
    $("#tail-connect").disabled = true;
    $("#tail-stop").disabled = false;

    ws.onopen = () => {
      if (state.tailWS !== ws) return;
      writeTailState({ ...collectTailFilters(), live: true });
      setTailStatus("live", "Live");
      if (!$(".tail-line", box)) {
        box.innerHTML = `<div class="tail-placeholder">Connected — waiting for signals…</div>`;
      }
    };
    ws.onmessage = (ev) => {
      try {
        appendTailLine(JSON.parse(ev.data));
      } catch {
        appendTailLine({ ts: new Date().toISOString(), level: "info", msg: ev.data });
      }
    };
    ws.onerror = () => {
      if (state.pageUnloading || state.tailStopping) return;
      setTailStatus("error", "Error");
    };
    ws.onclose = () => {
      if (state.tailWS === ws) state.tailWS = null;
      const wasStopping = state.tailStopping;
      state.tailStopping = false;
      if (state.pageUnloading) {
        // Refresh / navigate away: keep live=true so the next load can resume.
        return;
      }
      $("#tail-stop").disabled = true;
      $("#tail-connect").disabled = false;
      if (wasStopping) {
        const cur = $("#tail-status").dataset.state;
        if (cur !== "error" && cur !== "idle") setTailStatus("idle", "Stopped");
        return;
      }
      // Abnormal disconnect — do not auto-resume after the next refresh.
      writeTailState({ ...collectTailFilters(), live: false });
      const cur = $("#tail-status").dataset.state;
      if (cur !== "error") setTailStatus("error", "Disconnected");
      else setTailStatus("error", "Error");
    };
  }

  window.addEventListener("pagehide", () => {
    state.pageUnloading = true;
    if (state.tailWS) {
      writeTailState({ ...collectTailFilters(), live: true });
    }
  });
  window.addEventListener("beforeunload", () => {
    state.pageUnloading = true;
    if (state.tailWS) {
      writeTailState({ ...collectTailFilters(), live: true });
    }
  });
  window.addEventListener("pageshow", (e) => {
    state.pageUnloading = false;
    if (e.persisted && readTailState().live && !state.tailWS) {
      startTail({ restored: true });
    }
  });

  $("#tail-form").addEventListener("submit", (e) => {
    e.preventDefault();
    startTail();
  });

  $("#tail-stop").addEventListener("click", () => {
    stopTail({ intentional: true });
    setTailStatus("idle", "Stopped");
  });

  $("#tail-clear").addEventListener("click", () => {
    $("#tail-results").innerHTML = `<div class="tail-placeholder">Cleared.</div>`;
    state.tailCount = 0;
    updateTailCount();
  });

  $("#tail-autoscroll").addEventListener("change", () => {
    writeTailState({ ...collectTailFilters(), live: !!state.tailWS || !!readTailState().live });
  });

  /* —— Archives —— */
  function showToast(msg, ok) {
    const el = $("#archives-toast");
    el.textContent = msg;
    el.classList.remove("hidden", "ok", "err");
    el.classList.add(ok ? "ok" : "err");
    clearTimeout(showToast._t);
    showToast._t = setTimeout(() => el.classList.add("hidden"), 4000);
  }

  async function loadArchives() {
    const tbody = $("#archives-results");
    setWrapState("#archives-wrap", "loading");
    tbody.innerHTML = "";
    try {
      const data = await fetchJSON("/api/archives");
      const list = Array.isArray(data) ? data : [];
      if (!list.length) {
        setWrapState("#archives-wrap", "empty");
        return;
      }
      for (const a of list) {
        const tr = document.createElement("tr");
        const range = `${fmtTime(a.from)} → ${fmtTime(a.to)}`;
        tr.innerHTML = `
          <td class="mono">${escapeHtml(a.id || "—")}</td>
          <td>${escapeHtml(a.kind || "—")}</td>
          <td class="mono">${escapeHtml(range)}</td>
          <td class="mono">${escapeHtml(String(a.count ?? "—"))}</td>
          <td class="mono">${escapeHtml(fmtBytes(a.size_bytes))}</td>
          <td class="mono">${escapeHtml(fmtTime(a.created_at))}</td>
          <td><button type="button" class="btn danger restore-btn">Restore</button></td>
        `;
        $(".restore-btn", tr).addEventListener("click", async () => {
          const btn = $(".restore-btn", tr);
          btn.disabled = true;
          try {
            await fetchJSON("/api/archives/restore", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ id: a.id, overwrite: true }),
            });
            showToast(`Restore requested: ${a.id}`, true);
          } catch (e) {
            showToast(e.message || "Restore failed", false);
          } finally {
            btn.disabled = false;
          }
        });
        tbody.appendChild(tr);
      }
      setWrapState("#archives-wrap", "ready");
    } catch (e) {
      setWrapState("#archives-wrap", "error", e.message || "Failed to list archives");
    }
  }

  $("#metrics-refresh").addEventListener("click", () => loadMetrics());
  $("#probes-refresh").addEventListener("click", () => loadProbes());
  $("#configs-refresh").addEventListener("click", () => loadConfigs());
  $("#archives-refresh").addEventListener("click", () => loadArchives());

  /* —— Boot —— */
  tickClock();
  setInterval(tickClock, 1000);
  pollHealth();
  setInterval(pollHealth, 15000);
  $("#tail-results").innerHTML = `<div class="tail-placeholder">Connect to begin streaming live signals.</div>`;
  updateTailCount();

  const route = parseHash();
  const savedTail = readTailState();
  applyTailFilters(savedTail);

  if (route.type === "signal") {
    applyRoute();
    if (savedTail.live) startTail({ restored: true });
  } else {
    const startTab = savedTail.live ? "tail" : route.tab;
    state.loaded[startTab] = true;
    switchTab(startTab, savedTail.live ? true : !location.hash);
    if (savedTail.live) {
      startTail({ restored: true });
    } else if (startTab === "logs") searchLogs(true);
    else if (startTab === "metrics") loadMetrics();
    else if (startTab === "probes") loadProbes();
    else if (startTab === "configs") loadConfigs();
    else if (startTab === "archives") loadArchives();
  }
})();
