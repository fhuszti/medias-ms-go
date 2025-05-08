ALTER TABLE medias
    DROP COLUMN   failure_message,
    MODIFY COLUMN size_bytes INT NOT NULL;