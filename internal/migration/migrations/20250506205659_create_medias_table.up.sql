CREATE TABLE medias (
    id           BINARY(16)      NOT NULL PRIMARY KEY,
    object_key   VARCHAR(100)    NOT NULL,
    mime_type    VARCHAR(50)     NOT NULL,
    size_bytes   INT             NOT NULL,
    status       VARCHAR(50)     NOT NULL,
    metadata     JSON            NULL,
    created_at   DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at   DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;