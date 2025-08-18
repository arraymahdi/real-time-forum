ALTER TABLE posts
  DROP FOREIGN KEY fk_posts_group,
  DROP COLUMN group_id,
  DROP COLUMN privacy;
