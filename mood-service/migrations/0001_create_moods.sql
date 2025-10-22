-- Enable pgcrypto for UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS moods (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id) ON DELETE CASCADE,
  mood text NOT NULL, -- e.g., 'very good','good','so so','bad','angry'
  emoji text,         -- optional emoji representation
  note text,          -- optional note
  mood_date date DEFAULT CURRENT_DATE,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- trigger to update updated_at on change
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_moods_updated_at ON moods;
CREATE TRIGGER update_moods_updated_at
BEFORE UPDATE ON moods
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
