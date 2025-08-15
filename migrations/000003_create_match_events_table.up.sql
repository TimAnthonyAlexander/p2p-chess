CREATE TABLE match_events (
  match_id UUID NOT NULL REFERENCES matches(id),
  seq INT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('move', 'resign', 'draw_offer', 'draw_accept', 'clock_tick')),
  payload JSONB NOT NULL,
  side CHAR(1) CHECK (side IN ('w', 'b')),
  ts_client TIMESTAMPTZ,
  ts_server TIMESTAMPTZ DEFAULT NOW() NOT NULL,
  zobrist TEXT NOT NULL,
  sig BYTEA,
  valid BOOL NOT NULL DEFAULT TRUE,
  PRIMARY KEY (match_id, seq)
);