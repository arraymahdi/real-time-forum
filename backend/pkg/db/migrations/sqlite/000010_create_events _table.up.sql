CREATE TABLE events (
  event_id INT PRIMARY KEY AUTO_INCREMENT,
  group_id INT NOT NULL,
  creator_id INT NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  event_time DATETIME NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (group_id) REFERENCES groups(group_id),
  FOREIGN KEY (creator_id) REFERENCES users(user_id)
);
