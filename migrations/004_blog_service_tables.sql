CREATE TABLE IF NOT EXISTS blog_service.blogs ( 
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), 
    user_id uuid, -- reference auth_service.users.id 
    title text NOT NULL, 
    content text NOT NULL, 
    published boolean DEFAULT false, 
    created_at timestamptz DEFAULT now(), 
    updated_at timestamptz DEFAULT now() 
);