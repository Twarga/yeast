CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    done BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO todos (title, done) VALUES
    ('Learn Yeast', FALSE),
    ('Build a multi-VM lab', TRUE),
    ('Snapshot and reset', FALSE);
