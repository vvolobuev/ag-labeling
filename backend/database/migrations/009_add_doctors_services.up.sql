ALTER TABLE departments ADD COLUMN IF NOT EXISTS doctors_services JSONB DEFAULT '{}'::jsonb;

UPDATE departments 
SET doctors_services = '{"998573-malyshin": [87, 69]}'::jsonb 
WHERE id = 'ultrasound';