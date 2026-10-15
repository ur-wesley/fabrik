ALTER TABLE issue_notes ADD COLUMN seq BIGINT NOT NULL AUTO_INCREMENT UNIQUE;

INSERT INTO meta (name, value) VALUES ('schema_version', '2')
ON DUPLICATE KEY UPDATE value = VALUES(value);
