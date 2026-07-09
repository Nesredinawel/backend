CREATE TABLE IF NOT EXISTS blog_service.blogs ( 
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), 
    user_id uuid, 
    title text NOT NULL, 
    content text NOT NULL,
    excerpt text DEFAULT '',
    category text DEFAULT 'general',
    tags jsonb DEFAULT '[]'::jsonb,
    read_time int DEFAULT 1,
    published boolean DEFAULT false, 
    created_at timestamptz DEFAULT now(), 
    updated_at timestamptz DEFAULT now() 
);

CREATE INDEX IF NOT EXISTS idx_blogs_category ON blog_service.blogs (category);
CREATE INDEX IF NOT EXISTS idx_blogs_published ON blog_service.blogs (published);

CREATE TABLE IF NOT EXISTS blog_service.blogs_images (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id uuid NOT NULL REFERENCES blog_service.blogs(id) ON DELETE CASCADE,
    url text NOT NULL,
    caption text,
    created_at timestamptz DEFAULT now()
);