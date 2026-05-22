package tls_gatekeeper

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nossas/bonde-caddy/internal/database"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(TLSGatekeeper{})
	httpcaddyfile.RegisterHandlerDirective("tls_gatekeeper", parseCaddyfile)
}

// TLSGatekeeper verifica se um domínio está autorizado a receber certificado SSL
type TLSGatekeeper struct {
	DatabaseURL string `json:"database_url,omitempty"`
	Path        string `json:"path,omitempty"` // endpoint path, ex: /tls_verify

	logger *zap.Logger
	db     *sql.DB
}

func (TLSGatekeeper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.tls_gatekeeper",
		New: func() caddy.Module { return new(TLSGatekeeper) },
	}
}

func (g *TLSGatekeeper) Provision(ctx caddy.Context) error {
	g.logger = ctx.Logger()

	db, err := database.NewConnection(database.Config{URL: g.DatabaseURL})
	if err != nil {
		return fmt.Errorf("gatekeeper database connection failed: %v", err)
	}
	g.db = db

	g.logger.Info("TLS Gatekeeper provisioned",
		zap.String("path", g.Path))

	return nil
}

func (g *TLSGatekeeper) Validate() error {
	if g.db == nil {
		return fmt.Errorf("database connection not established")
	}
	return nil
}

func (g TLSGatekeeper) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Só processa no path configurado
	if g.Path != "" && r.URL.Path != g.Path {
		return next.ServeHTTP(w, r)
	}

	// Pega o domínio do parâmetro 'domain' (como o on_demand_tls envia)
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		// Fallback: tenta pegar do header
		domain = r.Header.Get("X-Forwarded-Host")
		if domain == "" {
			domain = r.Host
		}
	}

	// Remove porta se existir
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	g.logger.Info("Verifying TLS permission",
		zap.String("domain", domain))

	// Consulta o banco
	enabled, err := database.IsSSLEnabled(g.db, domain)
	if err != nil {
		g.logger.Error("Database query failed",
			zap.String("domain", domain),
			zap.Error(err))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": false,
			"error":   "database error",
		})
		return nil
	}

	// Responde no formato que o on_demand_tls espera
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if enabled {
		g.logger.Info("Domain authorized for TLS",
			zap.String("domain", domain))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": true,
			"domain":  domain,
		})
	} else {
		g.logger.Warn("Domain not authorized for TLS",
			zap.String("domain", domain))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": false,
			"domain":  domain,
		})
	}

	return nil
}

func (g *TLSGatekeeper) UnmarshalCaddyfile(dispenser *caddyfile.Dispenser) error {
	dispenser.Next()

	// O primeiro argumento é o path
	if dispenser.NextArg() {
		g.Path = dispenser.Val()
	}

	for dispenser.NextBlock(0) {
		switch dispenser.Val() {
		case "database_url":
			if !dispenser.NextArg() {
				return dispenser.ArgErr()
			}
			g.DatabaseURL = dispenser.Val()
		}
	}
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var g TLSGatekeeper
	err := g.UnmarshalCaddyfile(h.Dispenser)
	return &g, err
}

func (g *TLSGatekeeper) Cleanup() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

var (
	_ caddy.Provisioner           = (*TLSGatekeeper)(nil)
	_ caddy.Validator             = (*TLSGatekeeper)(nil)
	_ caddyhttp.MiddlewareHandler = (*TLSGatekeeper)(nil)
	_ caddyfile.Unmarshaler       = (*TLSGatekeeper)(nil)
	_ caddy.CleanerUpper          = (*TLSGatekeeper)(nil)
)