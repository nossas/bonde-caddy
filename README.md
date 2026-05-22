# BONDE Caddy

Módulos customizados para o [Caddy Server](https://caddyserver.com/) que adicionam roteamento dinâmico baseado em banco de dados e controle de TLS on-demand.

---

## 📦 Módulos

### 🔀 Domain Resolver
Proxy reverso dinâmico que resolve domínios para upstreams consultando uma tabela PostgreSQL.

**Diretiva Caddyfile:** `domain_resolver`

**Exemplo:**

```caddy
# Caddyfile

:80 {
	route {
		domain_resolver {
			database_url {$DATABASE_URL}
			cache_ttl 5m
		}
	}
}
```

### 🛡️ TLS Gatekeeper
Controla quais domínios podem receber certificados SSL/TLS via `on_demand_tls`, consultando o banco de dados.

**Diretiva Caddyfile:** `tls_gatekeeper`

**Exemplo:**

```caddy
# Caddyfile
{
	on_demand_tls {
		ask http://localhost:80/ask
	}
}

:80 {
	route {
		tls_gatekeeper /ask {
			database_url {$DATABASE_URL}
		}
	}
}

:443 {
	tls {
		on_demand
	}
}
```

---

## 🚀 Instalação

### Desenvolvimento Local

```bash
# Clone o repositório
git clone https://github.com/nossas/bonde-caddy.git
cd bonde-caddy

# Inicie os serviços
docker compose -f docker/docker-compose.yml up -d

# Acompanhe os logs
docker compose -f docker/docker-compose.yml logs -f
```

## ⚙️ Configuração

### Banco de Dados

Execute o SQL de inicialização:

```bash
psql -U caddy -d caddy_proxy -f config/init.sql
```

**Tabela** `proxy_routes`:

| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `domain` | VARCHAR(255) | Domínio da rota |
| `upstream` | VARCHAR(255) | Destino do proxy (host:porta) |
| `active` | BOOLEAN | Rota ativa/inativa |
| `ssl_enabled` | BOOLEAN | Permite TLS on-demand |

---

## 🎯 Exemplos de Uso

### Adicionar uma rota HTTP

```sql
INSERT INTO proxy_routes (domain, upstream, active, ssl_enabled) 
VALUES ('meuapp.localhost', 'nginx:80', true, false);
```

### Adicionar uma rota HTTPS

```sql
INSERT INTO proxy_routes (domain, upstream, active, ssl_enabled) 
VALUES ('seguro.localhost', 'nginx-app:80', true, true);
```

### Desativar uma rota

```sql
UPDATE proxy_routes SET active = false WHERE domain = 'meuapp.localhost';
```

### Habilitar SSL para rota existente

```sql
UPDATE proxy_routes SET ssl_enabled = true WHERE domain = 'meuapp.localhost';
```

---

## 🧪 Testando

### Verificar endpoint de TLS

```bash
# Domínio autorizado
curl "http://localhost/tls_verify?domain=seguro.localhost"
# {"allowed":true,"domain":"seguro.localhost"}

# Domínio não autorizado
curl "http://localhost/tls_verify?domain=meuapp.localhost"
# {"allowed":false,"domain":"meuapp.localhost"}
```

### Testar proxy HTTP

```bash
curl -H "Host: meuapp.localhost" http://localhost
```

### Testar proxy HTTPS

```bash
curl -k -H "Host: seguro.localhost" https://localhost
```

---

## 🏗️ Estrutura do Projeto

```text
bonde-caddy/
├── cmd/
│   └── bonde-caddy/          # Entry point do binário
│       └── main.go
├── pkg/
│   └── caddy/                # Módulos públicos
│       ├── domain_resolver/  # Roteador dinâmico
│       │   └── resolver.go
│       ├── tls_gatekeeper/   # Controlador TLS
│       │   └── gatekeeper.go
│       └── logger/           # Logger de IPs
│           └── logger.go
├── internal/
│   └── database/             # Camada de dados (privada)
│       └── postgres.go
├── config/                   # Arquivos de configuração
│   ├── Caddyfile
│   └── init.sql
├── docker/                   # Docker Compose
│   └── docker-compose.yml
├── go.mod
└── go.sum
```

---

## 🔧 Desenvolvimento

### Pré-requisitos

- Go 1.23+
- Docker e Docker Compose
- PostgreSQL 16

### Comandos úteis

```bash
# Iniciar ambiente
docker compose -f docker/docker-compose.yml up -d

# Ver logs
docker compose -f docker/docker-compose.yml logs -f caddy-dev

# Reiniciar Caddy
docker compose -f docker/docker-compose.yml restart caddy-dev

# Acessar banco de dados
docker compose -f docker/docker-compose.yml exec postgres psql -U caddy -d caddy_proxy

# Parar tudo
docker compose -f docker/docker-compose.yml down
```

---

## 📊 Cache

O `domain_resolver` possui cache em memória para reduzir consultas ao banco:

| Configuração | Padrão | Descrição |
|----|----|----|
| `cache_ttl` | `30s` | Tempo de vida do cache |

Exemplos: `30s`, `5m`, `1h`

---

## 🔒 Segurança

- Use variáveis de ambiente para `DATABASE_URL`
- Em produção, use Let's Encrypt em vez de `tls internal`
- Restrinja o endpoint `/ask` apenas para localhost