ALTER TABLE departments ADD COLUMN IF NOT EXISTS services_list INTEGER[] DEFAULT '{}';

UPDATE departments 
SET services_list = ARRAY[57,58,59,60,61,62,63,64,65,66,67,68]
WHERE id = 'ultrasound';