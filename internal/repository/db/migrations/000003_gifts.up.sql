CREATE TABLE gifts (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title              text NOT NULL,
    message            text NOT NULL DEFAULT '',
    pixel_art          jsonb NOT NULL,        -- { width, height, palette: [...], pixels: [...] }
    reveal_type        text NOT NULL,         -- box | envelope | scratch | puzzle | confetti | cake | arcade
    reveal_config      jsonb NOT NULL DEFAULT '{}'::jsonb,
    view_token         text UNIQUE NOT NULL,  -- opaco, se comparte en claro con el destinatario
    recipient_email    citext,
    scheduled_open_at  timestamptz,
    scheduled_send_at  timestamptz,
    sent_at            timestamptz,
    single_open        boolean NOT NULL DEFAULT false,
    opened_at          timestamptz,
    expires_at         timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- Listado de regalos por creador (dashboard): UserRepository/GiftRepository.ListByUser.
CREATE INDEX gifts_creator_id_idx ON gifts (creator_id);
