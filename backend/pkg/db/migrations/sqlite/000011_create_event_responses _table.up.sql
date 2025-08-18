CREATE TABLE event_responses (
  event_id INT,
  user_id INT,
  response ENUM('going','not_going') NOT NULL,
  responded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (event_id, user_id),
  FOREIGN KEY (event_id) REFERENCES events(event_id),
  FOREIGN KEY (user_id) REFERENCES users(user_id)
);
