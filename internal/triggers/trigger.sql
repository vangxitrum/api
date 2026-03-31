CREATE OR REPLACE FUNCTION delete_cdn_file()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM cdn_files WHERE id = OLD.file_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_delete_media_caption_files
AFTER DELETE ON media_caption_files
FOR EACH ROW
EXECUTE FUNCTION delete_cdn_file();

CREATE OR REPLACE TRIGGER trg_delete_media_chapter_files
AFTER DELETE ON media_chapter_files
FOR EACH ROW
EXECUTE FUNCTION delete_cdn_file();

CREATE OR REPLACE TRIGGER trg_delete_media_files
AFTER DELETE ON media_files
FOR EACH ROW
EXECUTE FUNCTION delete_cdn_file();

CREATE OR REPLACE TRIGGER trg_delete_media_quality_files
AFTER DELETE ON media_quality_files
FOR EACH ROW
EXECUTE FUNCTION delete_cdn_file();

CREATE OR REPLACE TRIGGER trg_delete_thumbnail_files
AFTER DELETE ON thumbnail_files
FOR EACH ROW
EXECUTE FUNCTION delete_cdn_file();

CREATE OR REPLACE FUNCTION delete_thumbnail()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM thumbnail_resolutions WHERE thumbnail_id = OLD.thumbnail_id;
    DELETE FROM thumbnail_files WHERE thumbnail_id = OLD.thumbnail_id;
    DELETE FROM thumbnails WHERE id = OLD.thumbnail_id;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_delete_media_thumbnails
AFTER DELETE ON media_thumbnails
FOR EACH ROW
EXECUTE FUNCTION delete_thumbnail();

CREATE OR REPLACE TRIGGER trg_delete_playlist_thumbnails
AFTER DELETE ON playlist_thumbnails
FOR EACH ROW
EXECUTE FUNCTION delete_thumbnail();

CREATE OR REPLACE FUNCTION public.delete_player_logo_file()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    DELETE FROM cdn_files WHERE id = OLD.player_asset_file_id;
    RETURN OLD;
END;
$function$
;

CREATE OR REPLACE TRIGGER trg_delete_player_logo
AFTER DELETE ON player_themes
FOR EACH ROW
EXECUTE FUNCTION delete_player_logo_file();


-- Unique index on email (excluding soft-deleted rows)
CREATE UNIQUE INDEX idx_users_email_active
  ON users (email)
  WHERE deleted_at IS NULL;

-- Unique index on wallet_connection (excluding soft-deleted rows)
CREATE UNIQUE INDEX idx_users_wallet_connection_active
  ON users (wallet_connection)
  WHERE deleted_at IS NULL and wallet_connection != '';



