CREATE TABLE IF NOT EXISTS positions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO positions (id, name) VALUES
('pediatrician', 'педиатр'),
('gynecologist_endocrinologist', 'гинеколог-эндокринолог'),
('ultrasound_doctor', 'врач УЗИ'),
('gynecologist', 'гинеколог'),
('mammologist', 'маммолог'),
('dermatologist', 'дерматолог'),
('venereologist', 'венеролог'),
('pediatric_dermatologist', 'детский дерматолог'),
('obstetrician', 'акушер'),
('physiotherapist', 'физиотерапевт'),
('cardiologist', 'кардиолог');