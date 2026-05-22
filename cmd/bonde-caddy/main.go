package main

import (
    caddycmd "github.com/caddyserver/caddy/v2/cmd"
    
    // Caddy standard modules
    _ "github.com/caddyserver/caddy/v2/modules/standard"
    
    // Bonde Caddy modules
    _ "github.com/nossas/bonde-caddy/pkg/caddy/domain_resolver"
    _ "github.com/nossas/bonde-caddy/pkg/caddy/tls_gatekeeper"
    // _ "github.com/nossas/bonde-caddy/pkg/caddy/logger"
)

func main() {
    caddycmd.Main()
}