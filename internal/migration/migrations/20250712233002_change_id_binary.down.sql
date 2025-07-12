-- Revert id from BINARY(16) back to VARCHAR(36)
ALTER TABLE medias ADD COLUMN id_new VARCHAR(36) NULL;

UPDATE medias SET id_new =
  LOWER(CONCAT(
    SUBSTR(HEX(id),1,8), '-',
    SUBSTR(HEX(id),9,4), '-',
    SUBSTR(HEX(id),13,4), '-',
    SUBSTR(HEX(id),17,4), '-',
    SUBSTR(HEX(id),21)
  ));

ALTER TABLE medias
    DROP PRIMARY KEY,
    DROP COLUMN id,
    CHANGE COLUMN id_new id VARCHAR(36) NOT NULL PRIMARY KEY;
