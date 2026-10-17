CREATE TABLE IF NOT EXISTS menu (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    icon TEXT,
    image_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- View services untuk kompatibilitas referensi layanan/services
-- CREATE VIEW IF NOT EXISTS services AS
-- SELECT id, name, description, icon, image_url, created_at
-- FROM menu;
