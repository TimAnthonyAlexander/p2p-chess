CREATE INDEX match_events_match_id_ts_server_idx ON match_events (match_id, ts_server);
CREATE INDEX matches_status_created_at_idx ON matches (status, created_at);
CREATE INDEX matches_side_white_idx ON matches (side_white);
CREATE INDEX matches_side_black_idx ON matches (side_black);