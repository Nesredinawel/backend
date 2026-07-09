-- ===============================
-- Schema: auth_service
-- Tables: users, user_profiles
-- ===============================

-- Users table
CREATE TABLE IF NOT EXISTS auth_service.users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text UNIQUE NOT NULL,
    name text NOT NULL,
    password text,            -- store hashed password
    avatar_url text,
    provider text,            -- e.g., "google", "email"
    provider_id text,         -- id from provider
    role text DEFAULT 'user', -- user role
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- User profiles table
CREATE TABLE IF NOT EXISTS auth_service.user_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid UNIQUE REFERENCES auth_service.users(id) ON DELETE CASCADE,
    bio text,
    custom_avatar_url text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);
