package bondecaddy

import (
	// Domain Resolver - dynamic proxy based on PostgreSQL
	_ "github.com/nossas/bonde-caddy/pkg/caddy/domain_resolver"

	// TLS Gatekeeper - controls on_demand_tls via database
	_ "github.com/nossas/bonde-caddy/pkg/caddy/tls_gatekeeper"
)