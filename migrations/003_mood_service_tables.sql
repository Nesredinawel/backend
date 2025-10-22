CREATE TABLE IF NOT EXISTS mood_service.moods (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid, -- reference auth_service.users.id
    mood text NOT NULL,
    emoji text,
    note text,
    mood_date date DEFAULT CURRENT_DATE,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);
