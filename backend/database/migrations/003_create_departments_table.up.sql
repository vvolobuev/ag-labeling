CREATE TABLE IF NOT EXISTS departments (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    short_description VARCHAR(200) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    position INTEGER,
    head_doctor_id VARCHAR(50)
);

INSERT INTO departments (id, name, description, short_description, is_active, position, head_doctor_id) VALUES
('gynecology', 'Гинекология', 'Гинекологическое отделение', 'Забота о женском здоровье', true, 1, NULL);