-- SQLite cannot ADD a UNIQUE column via ALTER TABLE, so seq arrives as a
-- plain INTEGER and the uniqueness invariant lives in a separate index
-- (Dolt uses AUTO_INCREMENT UNIQUE inline; see sql/dolt/002).
ALTER TABLE issue_notes ADD COLUMN seq INTEGER;

UPDATE issue_notes SET seq = rowid WHERE seq IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS issue_notes_seq_uq ON issue_notes(seq);

INSERT INTO meta (name, value) VALUES ('schema_version', '2')
ON CONFLICT(name) DO UPDATE SET value = excluded.value;
