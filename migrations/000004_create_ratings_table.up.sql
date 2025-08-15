CREATE TABLE ratings (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  rating REAL,
  rd REAL,
  volatility REAL,
  updated_at TIMESTAMPTZ
);