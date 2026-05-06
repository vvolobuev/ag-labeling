ALTER TABLE doctors ADD COLUMN IF NOT EXISTS achievements JSONB DEFAULT '[]'::jsonb;

UPDATE doctors SET achievements = '[
  {"title": "врач высшей категории", "position": 1}
]'::jsonb WHERE id = '8973-oflidi';

UPDATE doctors SET achievements = '[
  {"title": "врач высшей категории", "position": 1}
]'::jsonb WHERE id = '629306-vasileva';