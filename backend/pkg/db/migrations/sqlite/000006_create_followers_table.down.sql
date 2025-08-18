DROP TRIGGER IF EXISTS prevent_self_follow;
DROP INDEX IF EXISTS idx_followers_follower_id;
DROP INDEX IF EXISTS idx_followers_following_id;
DROP TABLE IF EXISTS followers;