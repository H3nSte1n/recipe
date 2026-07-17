-- Track consecutive failed login attempts and a lockout expiry so repeated bad
-- passwords against one account can be locked out temporarily (see UserService.Login).
ALTER TABLE users ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE;
