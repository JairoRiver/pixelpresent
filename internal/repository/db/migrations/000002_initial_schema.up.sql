CREATE TABLE users (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email       citext UNIQUE NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
