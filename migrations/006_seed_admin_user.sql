-- Seed admin user for blog management
-- Email: admin@goodmorning.app
-- Password: admin123

INSERT INTO auth_service.users (id, email, name, password, provider, role)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'admin@goodmorning.app',
  'Admin',
  '$2a$10$wFxPictJuohBfT7lMyyrh.U9gLSa1U9ATBFbEVZQ2x9rIr09dkdKK',
  'email',
  'admin'
)
ON CONFLICT (id) DO UPDATE SET
  role = 'admin',
  password = EXCLUDED.password;
