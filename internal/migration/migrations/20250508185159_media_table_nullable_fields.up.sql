ALTER TABLE medias
    MODIFY COLUMN size_bytes      INT  NULL,
    ADD COLUMN    failure_message TEXT NULL;
