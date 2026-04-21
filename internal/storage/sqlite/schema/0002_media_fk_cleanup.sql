-- 0002: clean up legacy media rows written by Phase 1 with empty card_id.
DELETE FROM media WHERE card_id = '' OR card_id IS NULL;
