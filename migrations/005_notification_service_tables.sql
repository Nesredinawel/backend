CREATE TABLE IF NOT EXISTS notification_service.notifications (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    title text NOT NULL,
    message text NOT NULL,
    source_service text,
    action text,
    meta jsonb,
    read boolean DEFAULT false,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_user_id_idx ON notification_service.notifications (user_id);
