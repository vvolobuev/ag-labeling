CREATE TABLE IF NOT EXISTS promotions (
    id SERIAL PRIMARY KEY,
    text TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO promotions (text, is_active) VALUES ('№ лицензии: ЛО-23-01-011698 (новая): ЛО-23-01-014467 от 25.03.20', true);