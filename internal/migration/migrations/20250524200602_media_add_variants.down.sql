ALTER TABLE medias
    DROP COLUMN variants,
    MODIFY COLUMN metadata JSON NULL;