ALTER TABLE metadata ADD COLUMN owner_id text DEFAULT '';
UPDATE metadata SET owner_id = '';

ALTER TABLE metadata ADD COLUMN modified TIMESTAMP DEFAULT now();
UPDATE metadata SET modified = now();