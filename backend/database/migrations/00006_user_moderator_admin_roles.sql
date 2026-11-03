-- +goose Up
ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'admin' WHERE role IN ('admin', 'superadmin');
ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'moderator', 'admin'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'user' WHERE role = 'moderator';
ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin', 'superadmin'));
