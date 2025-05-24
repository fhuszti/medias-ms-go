UPDATE medias
    SET metadata = '{}'
    WHERE metadata IS NULL;
ALTER TABLE medias
    MODIFY COLUMN metadata JSON NOT NULL DEFAULT '{}',
    ADD    COLUMN variants JSON NOT NULL DEFAULT '[]';