CREATE TABLE users (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email       citext UNIQUE NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE magic_links (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   text UNIQUE NOT NULL,   -- sha256(token), nunca el token en claro
    expires_at   timestamptz NOT NULL,
    consumed_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

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
    published_at       timestamptz,           -- NULL = borrador (oculto al destinatario); no NULL = publicado
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- Listado de regalos por creador (dashboard): GiftRepository.ListByUser.
CREATE INDEX gifts_creator_id_idx ON gifts (creator_id);

CREATE TABLE reactions (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    gift_id            uuid NOT NULL REFERENCES gifts(id) ON DELETE CASCADE,
    kind               text NOT NULL,         -- emoji | text | voice
    emoji              text,
    message            text,
    voice_storage_key  text,
    created_at         timestamptz NOT NULL DEFAULT now()
);

-- Listado de reacciones por regalo (vista del creador): ReactionRepository.ListByGift.
CREATE INDEX reactions_gift_id_idx ON reactions (gift_id);
