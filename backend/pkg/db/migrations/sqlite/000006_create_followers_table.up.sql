CREATE TABLE followers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    follower_id INTEGER NOT NULL,
    following_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (following_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(follower_id, following_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_followers_follower_id ON followers(follower_id);
CREATE INDEX idx_followers_following_id ON followers(following_id);

-- Prevent users from following themselves
CREATE TRIGGER prevent_self_follow
    BEFORE INSERT ON followers
    BEGIN
        SELECT CASE
            WHEN NEW.follower_id = NEW.following_id THEN
                RAISE(ABORT, 'Users cannot follow themselves')
        END;
    END;