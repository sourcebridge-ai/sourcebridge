CREATE TABLE IF NOT EXISTS pings (
  installation_id TEXT PRIMARY KEY,
  version TEXT NOT NULL DEFAULT 'unknown',
  edition TEXT NOT NULL DEFAULT 'oss',
  platform TEXT NOT NULL DEFAULT 'unknown',
  repos INTEGER NOT NULL DEFAULT 0,
  users INTEGER NOT NULL DEFAULT 0,
  features TEXT NOT NULL DEFAULT '[]',
  counts TEXT NOT NULL DEFAULT '{}',
  ping_count INTEGER NOT NULL DEFAULT 1,
  first_seen TEXT NOT NULL DEFAULT (datetime('now')),
  last_seen TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pings_last_seen ON pings(last_seen);
CREATE INDEX IF NOT EXISTS idx_pings_version ON pings(version);
CREATE INDEX IF NOT EXISTS idx_pings_edition ON pings(edition);
