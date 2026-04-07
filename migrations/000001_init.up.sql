-- 000001_init.up.sql
-- Initial schema for Where is Church?

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    denomination VARCHAR(100) NOT NULL DEFAULT 'Catholic',
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);

CREATE TABLE IF NOT EXISTS churches (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    denomination VARCHAR(100) NOT NULL DEFAULT 'Catholic',
    address VARCHAR(500) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    phone VARCHAR(50),
    website VARCHAR(500),
    description TEXT,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    last_verified TIMESTAMPTZ,
    created_by_id BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_churches_lat ON churches (latitude);
CREATE INDEX idx_churches_lng ON churches (longitude);
CREATE INDEX idx_churches_denomination ON churches (denomination);
CREATE INDEX idx_churches_verified ON churches (verified);

CREATE TABLE IF NOT EXISTS mass_schedules (
    id BIGSERIAL PRIMARY KEY,
    church_id BIGINT NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL DEFAULT 'mass',
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time VARCHAR(5) NOT NULL,
    end_time VARCHAR(5),
    language VARCHAR(100) DEFAULT 'English',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mass_schedules_church_id ON mass_schedules (church_id);
CREATE INDEX idx_mass_schedules_type ON mass_schedules (type);
CREATE INDEX idx_mass_schedules_day ON mass_schedules (day_of_week);

CREATE TABLE IF NOT EXISTS check_ins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    church_id BIGINT NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_check_ins_user_id ON check_ins (user_id);
CREATE INDEX idx_check_ins_church_id ON check_ins (church_id);
CREATE INDEX idx_check_ins_created_at ON check_ins (created_at);

CREATE TABLE IF NOT EXISTS suggestions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    church_id BIGINT REFERENCES churches(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    reviewed_by_id BIGINT REFERENCES users(id),
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_suggestions_user_id ON suggestions (user_id);
CREATE INDEX idx_suggestions_status ON suggestions (status);

CREATE TABLE IF NOT EXISTS favorites (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    church_id BIGINT NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, church_id)
);

CREATE INDEX idx_favorites_user_id ON favorites (user_id);
