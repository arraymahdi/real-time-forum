CREATE TABLE group_memberships (
  user_id INT,
  group_id INT,
  role ENUM('creator','member','admin') DEFAULT 'member',
  status ENUM('invited','pending','accepted','rejected') DEFAULT 'invited',
  joined_at TIMESTAMP NULL,
  PRIMARY KEY (user_id, group_id),
  FOREIGN KEY (user_id) REFERENCES users(user_id),
  FOREIGN KEY (group_id) REFERENCES groups(group_id)
);
