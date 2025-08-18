CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nickname TEXT UNIQUE,
    age INTEGER,
    gender TEXT,
    first_name TEXT,
    last_name TEXT,
    email TEXT UNIQUE,
    password TEXT
);
