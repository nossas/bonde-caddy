CREATE TABLE IF NOT EXISTS proxy_routes (
    id SERIAL PRIMARY KEY,
    domain VARCHAR(255) NOT NULL UNIQUE,
    upstream VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    ssl_enabled BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Dados de exemplo com SSL habilitado
INSERT INTO proxy_routes (domain, upstream, ssl_enabled) VALUES 
    ('localhost', 'nginx:80', false),
    ('site2.localhost', 'nginx-site2:80', true),
    ('secure.localhost', 'nginx:80', true)
ON CONFLICT (domain) DO NOTHING;