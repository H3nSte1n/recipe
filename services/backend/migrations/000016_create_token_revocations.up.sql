-- Tracks the newest time at which a user's previously issued JWTs must be
-- treated as invalid (e.g. after a password reset or account deletion). The
-- auth middleware rejects any token whose "iat" claim predates this
-- timestamp for the same user_id, without needing to track individual token
-- IDs.
--
-- No foreign key to users(id): a revocation must survive account deletion so
-- that any token issued before the deletion is still rejected after the
-- user row is gone.
CREATE TABLE token_revocations (
    user_id    UUID PRIMARY KEY,
    revoked_at TIMESTAMP WITH TIME ZONE NOT NULL
);
