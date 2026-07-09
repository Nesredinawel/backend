CREATE TABLE IF NOT EXISTS mood_service.moods (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid,
    mood text NOT NULL,
    mood_score integer NOT NULL DEFAULT 6,
    emoji text,
    note text,
    mood_date date DEFAULT CURRENT_DATE,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

ALTER TABLE mood_service.moods ADD CONSTRAINT moods_user_id_mood_date_idx UNIQUE (user_id, mood_date);
