FROM caddy:builder AS builder

COPY . /bonde-caddy

RUN xcaddy build \
    --with github.com/nossas/bonde-caddy=/bonde-caddy \
    --with github.com/caddy-dns/route53

# 
FROM caddy:2-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy