ALTER TABLE posts
    ADD COLUMN group_id INT,
    ADD COLUMN  privacy ENUM('public','almost_private','private') DEFAULT 'private',
    ADD FOREIGN KEY (group_id) REFERENCES groups(group_id);
