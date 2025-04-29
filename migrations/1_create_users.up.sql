DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin', 'manager');
    END IF;
END $$;

CREATE TABLE users (
   id SERIAL PRIMARY KEY,
   phone_number VARCHAR(20) UNIQUE NOT NULL,
   email VARCHAR(255) UNIQUE NOT NULL,
   name VARCHAR(100) NOT NULL,
   telegram VARCHAR(255),
   city VARCHAR(100),
   password_hash TEXT NOT NULL,
   referral_code VARCHAR(100) UNIQUE NOT NULL,
   created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
   is_active BOOLEAN DEFAULT false,
   roles user_role[] DEFAULT ARRAY['user'::user_role]
);

-- Это пользователь нужен для комментов. Через него нельзя будет зайти в учетку
INSERT INTO users (
    id,
    phone_number,
    email,
    name,
    telegram,
    city,
    password_hash,
    referral_code,
    is_active,
    roles
) VALUES (
    228,
    '+12345678901123',                            -- пример номера
    'manager@example.com',                   -- пример почты
    'Manager Name',                          -- имя
    '@manager_telegram',                     -- телеграм
    'Москва',                                -- город
    'hashed_password_here',                  -- сюда вставь хэш пароля
    'REFCODE123',                            -- уникальный реферальный код
    true,                                    -- активный
    ARRAY['manager'::user_role]              -- роль
);

