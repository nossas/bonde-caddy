package domain_resolver

import (
    "database/sql"
    "fmt"
    "io"
    "net/http"
    "strings"
    "sync"
    "time"
    
    "github.com/caddyserver/caddy/v2"
    "github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
    "github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
    "github.com/caddyserver/caddy/v2/modules/caddyhttp"
    "github.com/nossas/bonde-caddy/internal/database"
    "go.uber.org/zap"
)

func init() {
    caddy.RegisterModule(DomainResolver{})
    httpcaddyfile.RegisterHandlerDirective("domain_resolver", parseCaddyfile)
}

// CacheEntry represents a cached route
type CacheEntry struct {
    Upstream  string
    ExpiresAt time.Time
}

// DomainResolver resolves domains to upstreams using database
type DomainResolver struct {
    DatabaseURL string `json:"database_url,omitempty"`
    CacheTTL    string `json:"cache_ttl,omitempty"`
    
    logger *zap.Logger
    db     *sql.DB
    client *http.Client
    cache  sync.Map
    ttl    time.Duration
}

func (DomainResolver) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "http.handlers.domain_resolver",
        New: func() caddy.Module { return new(DomainResolver) },
    }
}

func (d *DomainResolver) Provision(ctx caddy.Context) error {
    d.logger = ctx.Logger()
    
    // Cache TTL
    if d.CacheTTL == "" {
        d.CacheTTL = "30s"
    }
    
    ttl, err := time.ParseDuration(d.CacheTTL)
    if err != nil {
        return fmt.Errorf("invalid cache_ttl: %v", err)
    }
    d.ttl = ttl
    
    // Database
    db, err := database.NewConnection(database.Config{URL: d.DatabaseURL})
    if err != nil {
        return err
    }
    d.db = db
    
    // HTTP Client
    d.client = &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        },
        Timeout: 30 * time.Second,
    }
    
    // Warm up cache
    go d.warmUpCache()
    
    d.logger.Info("Domain Resolver provisioned",
        zap.String("cache_ttl", d.CacheTTL))
    
    return nil
}

func (d *DomainResolver) Validate() error {
    if d.db == nil {
        return fmt.Errorf("database connection not established")
    }
    return nil
}

func (d DomainResolver) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
    host := r.Host
    if idx := strings.Index(host, ":"); idx != -1 {
        host = host[:idx]
    }
    
    // Prevent loops
    if r.Header.Get("X-Proxy-Handled") == "true" {
        return next.ServeHTTP(w, r)
    }
    
    // Check cache
    upstream, found := d.getFromCache(host)
    if !found {
        // Query database
        var err error
        upstream, err = database.GetUpstream(d.db, host)
        if err != nil {
            d.logger.Warn("No route found",
                zap.String("host", host),
                zap.Error(err))
            http.Error(w, "Not Found", http.StatusNotFound)
            return nil
        }
        
        // Store in cache
        d.setCache(host, upstream)
    }
    
    // Build target URL
    targetURL := fmt.Sprintf("http://%s%s", upstream, r.URL.Path)
    if r.URL.RawQuery != "" {
        targetURL += "?" + r.URL.RawQuery
    }
    
    // Create proxy request
    proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
    if err != nil {
        d.logger.Error("Failed to create proxy request", zap.Error(err))
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return nil
    }
    
    // Copy headers
    for key, values := range r.Header {
        if key == "Host" || key == "X-Proxy-Handled" {
            continue
        }
        for _, value := range values {
            proxyReq.Header.Add(key, value)
        }
    }
    
    proxyReq.Header.Set("X-Proxy-Handled", "true")
    proxyReq.Header.Set("X-Forwarded-Host", r.Host)
    proxyReq.Header.Set("X-Real-IP", r.RemoteAddr)
    
    // Execute request
    resp, err := d.client.Do(proxyReq)
    if err != nil {
        d.logger.Error("Proxy request failed", zap.Error(err))
        http.Error(w, "Bad Gateway", http.StatusBadGateway)
        return nil
    }
    defer resp.Body.Close()
    
    // Copy response
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
    
    return nil
}

// Cache methods
func (d *DomainResolver) getFromCache(host string) (string, bool) {
    value, exists := d.cache.Load(host)
    if !exists {
        return "", false
    }
    
    entry := value.(CacheEntry)
    if time.Now().After(entry.ExpiresAt) {
        d.cache.Delete(host)
        return "", false
    }
    
    return entry.Upstream, true
}

func (d *DomainResolver) setCache(host, upstream string) {
    entry := CacheEntry{
        Upstream:  upstream,
        ExpiresAt: time.Now().Add(d.ttl),
    }
    d.cache.Store(host, entry)
}

func (d *DomainResolver) warmUpCache() {
    // Implementation here
}

func (d *DomainResolver) UnmarshalCaddyfile(dispenser *caddyfile.Dispenser) error {
    dispenser.Next()
    
    for dispenser.NextBlock(0) {
        switch dispenser.Val() {
        case "database_url":
            if !dispenser.NextArg() {
                return dispenser.ArgErr()
            }
            d.DatabaseURL = dispenser.Val()
        case "cache_ttl":
            if !dispenser.NextArg() {
                return dispenser.ArgErr()
            }
            d.CacheTTL = dispenser.Val()
        }
    }
    return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
    var d DomainResolver
    err := d.UnmarshalCaddyfile(h.Dispenser)
    return &d, err
}

func (d *DomainResolver) Cleanup() error {
    if d.db != nil {
        return d.db.Close()
    }
    return nil
}

var (
    _ caddy.Provisioner           = (*DomainResolver)(nil)
    _ caddy.Validator             = (*DomainResolver)(nil)
    _ caddyhttp.MiddlewareHandler = (*DomainResolver)(nil)
    _ caddyfile.Unmarshaler       = (*DomainResolver)(nil)
    _ caddy.CleanerUpper          = (*DomainResolver)(nil)
)