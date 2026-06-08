const state = {
  authMode: "login",
  token: localStorage.getItem("cex.token") || "",
  user: JSON.parse(localStorage.getItem("cex.user") || "null"),
  markets: [],
  selectedMarket: null,
  orderSide: "buy",
  socket: null,
};

const $ = (id) => document.getElementById(id);

const els = {
  logoutBtn: $("logoutBtn"),
  sessionBox: $("sessionBox"),
  adminSetupPanel: $("adminSetupPanel"),
  authForm: $("authForm"),
  authSubmit: $("authSubmit"),
  emailInput: $("emailInput"),
  passwordInput: $("passwordInput"),
  balancesList: $("balancesList"),
  refreshBalancesBtn: $("refreshBalancesBtn"),
  refreshPortfolioBtn: $("refreshPortfolioBtn"),
  portfolioValue: $("portfolioValue"),
  portfolioExposure: $("portfolioExposure"),
  tradesList: $("tradesList"),
  depositForm: $("depositForm"),
  depositAsset: $("depositAsset"),
  depositAmount: $("depositAmount"),
  assetForm: $("assetForm"),
  assetSymbol: $("assetSymbol"),
  assetName: $("assetName"),
  marketForm: $("marketForm"),
  marketName: $("marketName"),
  marketBase: $("marketBase"),
  marketQuote: $("marketQuote"),
  marketMin: $("marketMin"),
  marketMax: $("marketMax"),
  marketMaker: $("marketMaker"),
  marketTaker: $("marketTaker"),
  marketsList: $("marketsList"),
  refreshMarketsBtn: $("refreshMarketsBtn"),
  selectedMarketLabel: $("selectedMarketLabel"),
  refreshBookBtn: $("refreshBookBtn"),
  sellBook: $("sellBook"),
  buyBook: $("buyBook"),
  orderForm: $("orderForm"),
  orderType: $("orderType"),
  orderPrice: $("orderPrice"),
  orderQuantity: $("orderQuantity"),
  orderSummary: $("orderSummary"),
  submitOrderBtn: $("submitOrderBtn"),
  statusPill: $("statusPill"),
  wsStatus: $("wsStatus"),
  eventFeed: $("eventFeed"),
  toast: $("toast"),
};

function fmt(value, digits = 8) {
  const n = Number(value || 0);
  return Number.isFinite(n)
    ? n.toLocaleString(undefined, { maximumFractionDigits: digits })
    : "0";
}

function money(value) {
  return `$${fmt(value, 2)}`;
}

function shortId(id = "") {
  return id.length > 12 ? `${id.slice(0, 8)}...${id.slice(-4)}` : id;
}

function setStatus(text) {
  els.statusPill.textContent = text;
}

function toast(message, isError = false) {
  els.toast.textContent = message;
  els.toast.style.borderColor = isError ? "var(--sell)" : "var(--line)";
  els.toast.classList.remove("hidden");
  window.clearTimeout(toast.timer);
  toast.timer = window.setTimeout(() => els.toast.classList.add("hidden"), 3200);
}

async function api(path, options = {}) {
  setStatus("API busy");
  const headers = { "Content-Type": "application/json", ...(options.headers || {}) };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;

  const res = await fetch(path, { ...options, headers });
  const text = await res.text();
  const data = text ? JSON.parse(text) : {};
  setStatus("API ready");

  if (!res.ok) {
    throw new Error(data.message || data.error || `Request failed: ${res.status}`);
  }
  return data;
}

function saveSession(payload) {
  state.token = payload.token;
  state.user = { id: payload.id, email: payload.email, role: payload.role || "user" };
  localStorage.setItem("cex.token", state.token);
  localStorage.setItem("cex.user", JSON.stringify(state.user));
  renderSession();
}

function clearSession() {
  state.token = "";
  state.user = null;
  localStorage.removeItem("cex.token");
  localStorage.removeItem("cex.user");
  renderSession();
  renderBalances([]);
  renderPortfolio(null);
  renderTrades([]);
}

function renderSession() {
  if (!state.user) {
    els.sessionBox.textContent = "Not signed in";
    els.logoutBtn.classList.add("hidden");
    els.adminSetupPanel.classList.add("hidden");
    return;
  }
  els.sessionBox.textContent = `${state.user.email} (${state.user.role || "user"})`;
  els.logoutBtn.classList.remove("hidden");
  els.adminSetupPanel.classList.toggle("hidden", state.user.role !== "admin");
}

function renderMarkets() {
  if (!state.markets.length) {
    els.marketsList.textContent = "No markets yet. Create assets and a market from Admin Setup.";
    els.marketsList.classList.add("muted");
    return;
  }
  els.marketsList.classList.remove("muted");
  els.marketsList.innerHTML = "";

  state.markets.forEach((market) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = `market-card ${state.selectedMarket?.id === market.id ? "active" : ""}`;
    btn.innerHTML = `
      <strong>${market.name}</strong>
      <span>${market.base_asset}/${market.quote_asset}</span>
      <span class="muted">Price ${fmt(market.current_price)} | Fee ${fmt(market.maker_fee, 2)}/${fmt(market.taker_fee, 2)}</span>
      <span class="muted">${shortId(market.id)}</span>
    `;
    btn.addEventListener("click", () => selectMarket(market.id));
    els.marketsList.appendChild(btn);
  });
}

function renderBalances(balances) {
  if (!state.token) {
    els.balancesList.textContent = "Sign in to view balances.";
    els.balancesList.classList.add("muted");
    return;
  }
  if (!balances.length) {
    els.balancesList.textContent = "No balances yet. Deposit an asset to begin.";
    els.balancesList.classList.add("muted");
    return;
  }
  els.balancesList.classList.remove("muted");
  els.balancesList.innerHTML = "";
  balances.forEach((balance) => {
    const row = document.createElement("div");
    row.className = "balance-row";
    row.innerHTML = `
      <strong>${balance.asset}</strong>
      <span>Available ${fmt(balance.available)}<br><span class="muted">Locked ${fmt(balance.locked)}</span></span>
    `;
    els.balancesList.appendChild(row);
  });
}

function renderPortfolio(portfolio) {
  if (!state.token) {
    els.portfolioValue.textContent = "$0.00";
    els.portfolioExposure.textContent = "$0.00";
    return;
  }
  els.portfolioValue.textContent = money(portfolio?.total_usd || 0);
  els.portfolioExposure.textContent = money(portfolio?.open_exposure || 0);
}

function renderTrades(trades) {
  if (!state.token) {
    els.tradesList.textContent = "Sign in to view trades.";
    els.tradesList.classList.add("muted");
    return;
  }
  if (!trades.length) {
    els.tradesList.textContent = "No trades yet.";
    els.tradesList.classList.add("muted");
    return;
  }
  els.tradesList.classList.remove("muted");
  els.tradesList.innerHTML = "";
  trades.forEach((trade) => {
    const row = document.createElement("div");
    row.className = "trade-row";
    row.innerHTML = `
      <strong class="${trade.side === "buy" ? "buy" : "sell"}">${trade.side.toUpperCase()}</strong>
      <span>
        ${fmt(trade.quantity)} ${trade.base_asset} @ ${fmt(trade.price)} ${trade.quote_asset}
        <br><span class="muted">${new Date(trade.created_at).toLocaleString()} | ${shortId(trade.id)}</span>
      </span>
    `;
    els.tradesList.appendChild(row);
  });
}

function renderBook(book) {
  const sells = book?.sells || book?.Sells || [];
  const buys = book?.buys || book?.Buys || [];
  renderBookSide(els.sellBook, sells, "sell");
  renderBookSide(els.buyBook, buys, "buy");
}

function renderBookSide(container, orders, side) {
  if (!orders.length) {
    container.textContent = side === "buy" ? "No buy orders." : "No sell orders.";
    container.classList.add("muted");
    return;
  }
  container.classList.remove("muted");
  container.innerHTML = "";
  orders.forEach((order) => {
    const row = document.createElement("div");
    row.className = "book-row";
    const remaining = Number(order.quantity || 0) - Number(order.filled_quantity || 0);
    row.innerHTML = `
      <span>${fmt(order.price)}</span>
      <span>${fmt(remaining)}</span>
      <span class="muted">${shortId(order.id)}</span>
    `;
    container.appendChild(row);
  });
}

function updateOrderSummary() {
  const market = state.selectedMarket;
  if (!market) {
    els.orderSummary.textContent = "Select a market to begin.";
    els.submitOrderBtn.textContent = `Place ${state.orderSide === "buy" ? "Buy" : "Sell"} Order`;
    return;
  }
  const price = Number(els.orderPrice.value || 0);
  const quantity = Number(els.orderQuantity.value || 0);
  const total = price * quantity;
  const lockAsset = state.orderSide === "buy" ? market.quote_asset : market.base_asset;
  const lockAmount = state.orderSide === "buy" ? total : quantity;
  els.orderSummary.textContent = `Market ${market.base_asset}/${market.quote_asset}. Locks ${fmt(lockAmount)} ${lockAsset}.`;
  els.submitOrderBtn.textContent = `Place ${state.orderSide === "buy" ? "Buy" : "Sell"} Order`;
}

function addFeed(event, tone = "") {
  if (els.eventFeed.classList.contains("muted")) {
    els.eventFeed.classList.remove("muted");
    els.eventFeed.innerHTML = "";
  }
  const row = document.createElement("div");
  row.className = "feed-row";
  row.style.borderColor = tone === "error" ? "var(--sell)" : tone === "ok" ? "var(--buy)" : "var(--line)";
  row.textContent = `[${new Date().toLocaleTimeString()}] ${event}`;
  els.eventFeed.prepend(row);
}

async function loadMarkets() {
  const data = await api("/markets");
  state.markets = data.markets || [];
  if (!state.selectedMarket && state.markets.length) state.selectedMarket = state.markets[0];
  renderMarkets();
  if (state.selectedMarket) await selectMarket(state.selectedMarket.id);
}

async function loadBalances() {
  if (!state.token) return renderBalances([]);
  const data = await api("/balances");
  renderBalances(data.balances || []);
}

async function loadPortfolio() {
  if (!state.token) {
    renderPortfolio(null);
    renderTrades([]);
    return;
  }
  const [portfolioData, tradesData] = await Promise.all([
    api("/portfolio"),
    api("/trades"),
  ]);
  renderPortfolio(portfolioData.portfolio);
  renderTrades(tradesData.trades || []);
}

async function loadOrderBook() {
  if (!state.selectedMarket) return;
  const data = await api(`/order_book/${state.selectedMarket.id}`);
  renderBook(data.order_book);
}

async function selectMarket(id) {
  const market = state.markets.find((m) => m.id === id);
  if (!market) return;
  state.selectedMarket = market;
  els.selectedMarketLabel.textContent = `${market.name} (${market.base_asset}/${market.quote_asset})`;
  renderMarkets();
  updateOrderSummary();
  await loadOrderBook();
  connectSocket();
}

function connectSocket() {
  if (state.socket) state.socket.close();
  if (!state.selectedMarket) return;

  const protocol = location.protocol === "https:" ? "wss" : "ws";
  const url = `${protocol}://${location.host}/ws/${state.selectedMarket.id}`;
  const socket = new WebSocket(url);
  state.socket = socket;
  els.wsStatus.textContent = "Socket connecting";

  socket.addEventListener("open", () => {
    els.wsStatus.textContent = `Socket live ${shortId(state.selectedMarket.id)}`;
    addFeed(`connected ${url}`, "ok");
  });
  socket.addEventListener("message", async (event) => {
    addFeed(event.data, "ok");
    try {
      await Promise.all([loadOrderBook(), loadBalances(), loadPortfolio()]);
    } catch (err) {
      addFeed(`realtime refresh failed: ${err.message}`, "error");
    }
  });
  socket.addEventListener("close", () => {
    els.wsStatus.textContent = "Socket disconnected";
  });
  socket.addEventListener("error", () => {
    els.wsStatus.textContent = "Socket error";
    addFeed("websocket error", "error");
  });
}

function bindEvents() {
  document.querySelectorAll("[data-auth-tab]").forEach((btn) => {
    btn.addEventListener("click", () => {
      state.authMode = btn.dataset.authTab;
      document.querySelectorAll("[data-auth-tab]").forEach((b) => b.classList.toggle("active", b === btn));
      els.authSubmit.textContent = state.authMode === "login" ? "Sign in" : "Register";
    });
  });

  document.querySelectorAll("[data-side]").forEach((btn) => {
    btn.addEventListener("click", () => {
      state.orderSide = btn.dataset.side;
      document.querySelectorAll("[data-side]").forEach((b) => b.classList.toggle("active", b === btn));
      updateOrderSummary();
    });
  });

  [els.orderPrice, els.orderQuantity, els.orderType].forEach((el) => el.addEventListener("input", updateOrderSummary));
  els.logoutBtn.addEventListener("click", clearSession);
  els.refreshMarketsBtn.addEventListener("click", safe(loadMarkets));
  els.refreshBalancesBtn.addEventListener("click", safe(loadBalances));
  els.refreshPortfolioBtn.addEventListener("click", safe(loadPortfolio));
  els.refreshBookBtn.addEventListener("click", safe(loadOrderBook));

  els.authForm.addEventListener("submit", safe(async (event) => {
    event.preventDefault();
    const body = JSON.stringify({ email: els.emailInput.value.trim(), password: els.passwordInput.value });
    if (state.authMode === "register") {
      await api("/register", { method: "POST", body });
      toast("Account created. Signing in...");
    }
    const login = await api("/login", { method: "POST", body });
    saveSession(login);
    await Promise.all([loadBalances(), loadPortfolio()]);
    toast("Signed in");
  }));

  els.depositForm.addEventListener("submit", safe(async (event) => {
    event.preventDefault();
    await api("/deposit", {
      method: "POST",
      body: JSON.stringify({ asset: els.depositAsset.value.trim().toUpperCase(), amount: Number(els.depositAmount.value) }),
    });
    els.depositAmount.value = "";
    await Promise.all([loadBalances(), loadPortfolio()]);
    toast("Deposit complete");
  }));

  els.assetForm.addEventListener("submit", safe(async (event) => {
    event.preventDefault();
    await api("/assets", {
      method: "POST",
      body: JSON.stringify({ symbol: els.assetSymbol.value.trim().toUpperCase(), name: els.assetName.value.trim() }),
    });
    els.assetForm.reset();
    toast("Asset created");
  }));

  els.marketForm.addEventListener("submit", safe(async (event) => {
    event.preventDefault();
    await api("/market", {
      method: "POST",
      body: JSON.stringify({
        name: els.marketName.value.trim(),
        base_asset: els.marketBase.value.trim().toUpperCase(),
        quote_asset: els.marketQuote.value.trim().toUpperCase(),
        min_order_size: Number(els.marketMin.value),
        max_order_size: Number(els.marketMax.value),
        maker_fee: Number(els.marketMaker.value),
        taker_fee: Number(els.marketTaker.value),
      }),
    });
    els.marketForm.reset();
    await loadMarkets();
    toast("Market created");
  }));

  els.orderForm.addEventListener("submit", safe(async (event) => {
    event.preventDefault();
    if (!state.selectedMarket) throw new Error("Select a market first");
    await api("/orders", {
      method: "POST",
      body: JSON.stringify({
        market_id: state.selectedMarket.id,
        order_type: els.orderType.value,
        side: state.orderSide,
        price: Number(els.orderPrice.value),
        quantity: Number(els.orderQuantity.value),
        base_asset: state.selectedMarket.base_asset,
        quote_asset: state.selectedMarket.quote_asset,
      }),
    });
    els.orderForm.reset();
    updateOrderSummary();
    await Promise.all([loadOrderBook(), loadBalances(), loadPortfolio()]);
    toast("Order placed");
  }));
}

function safe(fn) {
  return async (...args) => {
    try {
      await fn(...args);
    } catch (err) {
      setStatus("API error");
      toast(err.message, true);
      addFeed(err.message, "error");
    }
  };
}

async function init() {
  bindEvents();
  renderSession();
  updateOrderSummary();
  await safe(loadMarkets)();
  await safe(loadBalances)();
  await safe(loadPortfolio)();
}

init();
