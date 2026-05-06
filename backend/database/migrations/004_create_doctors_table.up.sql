CREATE TABLE IF NOT EXISTS doctors (
    id VARCHAR(50) PRIMARY KEY,
    last_name VARCHAR(100) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    patronymic VARCHAR(100) NOT NULL,
    positions TEXT[] NOT NULL,
    departments TEXT[] NOT NULL,
    is_active BOOLEAN DEFAULT true,
    available_for_appointment BOOLEAN DEFAULT true,
    position INTEGER,
    graduation_date TIMESTAMP,
    has_red_diploma BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS doctor_education (
    doctor_id VARCHAR(50),
    institution VARCHAR(300) NOT NULL,
    graduation_year INTEGER NOT NULL,
    specialty VARCHAR(200),
    education_type VARCHAR(50) NOT NULL
);

INSERT INTO doctors (id, last_name, first_name, patronymic, positions, departments, is_active, available_for_appointment, position, graduation_date, has_red_diploma) VALUES
('629306-vasileva', 'Васильева', 'Галина', 'Борисовна', '{"pediatrician"}', '{"pediatrics"}', true, true, 1, '1981-01-01', true),
('283136-leonteva', 'Леонтьева', 'Наталья', 'Владимировна', '{"gynecologist_endocrinologist", "ultrasound_doctor", "gynecologist", "mammologist"}', '{"gynecology"}', true, true, 2, '2000-01-01', false),
('301580-pashyan', 'Пашян', 'Левон', 'Карапетович', '{"dermatologist", "venereologist", "pediatric_dermatologist"}', '{"dermatology"}', true, true, 3, '1981-01-01', false),
('629305-bedova', 'Бедова', 'Анна', 'Игоревна', '{"obstetrician", "ultrasound_doctor", "gynecologist", "physiotherapist"}', '{"gynecology"}', true, true, 4, '1995-01-01', true),
('865548-prokopov', 'Прокопов', 'Владимир', 'Евгеньевич', '{"dermatologist", "venereologist", "pediatric_dermatologist"}', '{"dermatology"}', true, true, 5, '1996-01-01', false);
