CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    USD_balance    NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (USD_balance >= 0),
    locked_balance NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (locked_balance >= 0),
    role          TEXT        NOT NULL CHECK (role IN ('user', 'admin')) DEFAULT 'user'
);
CREATE TABLE assets(
    symbol        TEXT        PRIMARY KEY,
    name          TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    icon_url      TEXT        NULL
);

CREATE TABLE markets(
    id           TEXT        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    base_asset   TEXT        NOT NULL REFERENCES assets(symbol) ON DELETE CASCADE,
    quote_asset  TEXT        NOT NULL REFERENCES assets(symbol) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    min_order_size NUMERIC(28,8) NOT NULL CHECK (min_order_size > 0),
    max_order_size NUMERIC(28,8) NOT NULL CHECK (max_order_size > 0),
    taker_fee     NUMERIC(5,2)  NOT NULL CHECK (taker_fee >= 0),
    maker_fee     NUMERIC(5,2)  NOT NULL CHECK (maker_fee >= 0),
    current_price   NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (current_price >= 0),
    UNIQUE (base_asset, quote_asset),
    
)

 
CREATE TABLE balances (
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset     TEXT        NOT NULL REFERENCES assets(symbol) ON DELETE CASCADE,
    available NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (available >= 0),
    locked    NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (locked    >= 0),
    PRIMARY KEY (user_id, asset)
);
CREATE TABLE orders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    market_id     TEXT        NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    quantity      NUMERIC(28,8) NOT NULL CHECK (quantity > 0),
    price         NUMERIC(28,8)  CHECK (price >= 0), --price can be 0 for market orders
    side          TEXT        NOT NULL CHECK (side IN ('buy', 'sell')),
    status        TEXT        NOT NULL CHECK (status IN ('open', 'filled', 'cancelled', 'pending')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type          TEXT        NOT NULL CHECK (type IN ('limit', 'market'))
    filled_quantity NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (filled_quantity >= 0),
    UNIQUE (user_id, market_id, created_at)
);
CREATE TABLE trades (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buy_order_id  UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    sell_order_id UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    quantity      NUMERIC(28,8) NOT NULL CHECK (quantity > 0),
    price         NUMERIC(28,8) NOT NULL CHECK (price > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    quote_asset TEXT        NOT NULL REFERENCES assets(symbol) ON DELETE CASCADE,
    base_asset TEXT        NOT NULL REFERENCES assets(symbol) ON DELETE CASCADE
);
CREATE TABLE transactions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    market_id     TEXT        NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    amount        NUMERIC(28,8) NOT NULL CHECK (amount != 0),
    type          TEXT        NOT NULL CHECK (type IN ('deposit', 'withdrawal')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE candles (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id   TEXT          NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    interval    TEXT          NOT NULL CHECK (interval IN ('1m', '5m', '15m', '1h', '4h', '1d')),
    open_time   TIMESTAMPTZ   NOT NULL,
    open_price  NUMERIC(28,8) NOT NULL CHECK (open_price > 0),
    high_price  NUMERIC(28,8) NOT NULL CHECK (high_price > 0),
    low_price   NUMERIC(28,8) NOT NULL CHECK (low_price > 0),
    close_price NUMERIC(28,8) NOT NULL CHECK (close_price > 0),
    volume      NUMERIC(28,8) NOT NULL DEFAULT 0 CHECK (volume >= 0),
    trade_count BIGINT        NOT NULL DEFAULT 1,
    UNIQUE (market_id, interval, open_time)
);
CREATE INDEX idx_candles_market_interval_time ON candles (market_id, interval, open_time DESC);
