package local

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/rilldata/rill/cli/cmd/env"
	"github.com/rilldata/rill/bratrax"
	"github.com/rilldata/rill/cli/pkg/browser"
	"github.com/rilldata/rill/cli/pkg/cmdutil"
	"github.com/rilldata/rill/cli/pkg/pkce"
	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/drivers"
	"github.com/rilldata/rill/runtime/parser"
	"github.com/rilldata/rill/runtime/pkg/activity"
	"github.com/rilldata/rill/runtime/pkg/debugserver"
	"github.com/rilldata/rill/runtime/pkg/email"
	"github.com/rilldata/rill/runtime/pkg/graceful"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"github.com/rilldata/rill/runtime/pkg/ratelimit"
	runtimeserver "github.com/rilldata/rill/runtime/server"
	"github.com/rilldata/rill/runtime/storage"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/structpb"
)

// Default instance config on local.
const (
	DefaultInstanceID   = "default"
	DefaultCatalogStore = "meta.db"
	DefaultDBDir        = "tmp"
)

// defaultMCPEnsureReadyTimeout bounds how long a /bratrax/mcp request waits for a
// cold instance's controller to become ready before returning a retryable 503.
// Kept below typical MCP client request timeouts so the wait resolves as a slow
// 200, not a client-side timeout. Override with
// BRATRAX_MCP_ENSURE_READY_TIMEOUT_SECONDS.
const defaultMCPEnsureReadyTimeout = 8 * time.Second

// defaultClaudeTemperature is the value the claude driver used to apply as its
// own default. The driver now omits `temperature` unless a connector asks for
// one (newer models 400 on it), so we set it explicitly here to keep clients on
// the default model answering exactly as they did before.
const defaultClaudeTemperature = 0.1

// App encapsulates the logic associated with configuring and running the UI and the runtime in a local environment.
// Here, a local environment means a non-authenticated, single-instance and single-project setup on localhost.
// App encapsulates logic shared between different CLI commands, like start, init, build and source.
type App struct {
	Context               context.Context
	Runtime               *runtime.Runtime
	Instance              *drivers.Instance
	Logger                *zap.SugaredLogger
	BaseLogger            *zap.Logger
	Verbose               bool
	Debug                 bool
	ProjectPath           string
	ch                    *cmdutil.Helper
	observabilityShutdown observability.ShutdownFunc
	loggerCleanUp         func()
	pkceAuthenticators    map[string]*pkce.Authenticator // map of state to pkce authenticators
	localURL              string
	allowedOrigins        []string
	// Multi-tenant fields. When MultiTenant is true, no default instance is created at startup;
	// instead per-client instances are created on-demand by EnsureInstanceForClient.
	MultiTenant bool
	ProjectsDir string                 // base directory containing per-client project dirs
	environment string                 // saved for instance creation
	debugFlag   bool                   // saved for instance creation
	instMu      sync.Mutex             // guards instances
	instances   map[string]*drivers.Instance
}

type AppOptions struct {
	Ch             *cmdutil.Helper
	Verbose        bool
	Silent         bool
	Debug          bool
	Reset          bool
	PullEnv        bool
	Environment    string
	ProjectPath    string
	LogFormat      string
	Variables      map[string]string
	LocalURL       string
	AllowedOrigins []string
	ServeUI        bool
	MultiTenant    bool   // Bratrax: when true, skip creating a default instance; instances created per-user
	ProjectsDir    string // Bratrax: base dir for per-client project dirs (multi-tenant mode)
}

func NewApp(ctx context.Context, opts *AppOptions) (*App, error) {
	// Skip project-path-bound setup when in multi-tenant mode (no default project).
	if !opts.MultiTenant {
		// Check that projectPath doesn't have an excessive number of files.
		// Note: Relies on ListGlob enforcing drivers.RepoListLimit.
		if _, err := os.Stat(opts.ProjectPath); err == nil {
			repo, _, err := cmdutil.RepoForProjectPath(opts.ProjectPath)
			if err != nil {
				return nil, err
			}
			_, err = repo.ListGlob(ctx, "**", false)
			if err != nil {
				if errors.Is(err, drivers.ErrRepoListLimitExceeded) {
					opts.Ch.PrintfError("The project directory exceeds the limit of %d files. Please open Rill against a directory with fewer files or set \"ignore_paths\" in rill.yaml.\n", drivers.RepoListLimit)
					return nil, nil
				}
				return nil, fmt.Errorf("failed to list project files: %w", err)
			}
		}

		// Always attempt to pull env for any valid Rill project (after projectPath is set)
		if opts.PullEnv && opts.Ch.IsAuthenticated() && IsProjectInit(opts.ProjectPath) {
			err := env.PullVars(ctx, opts.Ch, opts.ProjectPath, "", opts.Environment, false)
			if err != nil && !errors.Is(err, cmdutil.ErrNoMatchingProject) {
				opts.Ch.PrintfWarn("Warning: failed to pull environment credentials: %v\n", err)
			}
		}
	}

	// Parse log format
	parsedLogFormat, ok := ParseLogFormat(opts.LogFormat)
	if !ok {
		return nil, fmt.Errorf("invalid log format %q", opts.LogFormat)
	}

	// Setup logger
	logPath, err := opts.Ch.DotRill.ResolveFilename("rill.log", true)
	if err != nil {
		return nil, err
	}
	logger, cleanupFn := initLogger(opts.Verbose, opts.Silent, parsedLogFormat, logPath)
	sugarLogger := logger.Sugar()

	var tracesExporter observability.Exporter
	if opts.Debug {
		tracesExporter = observability.FileBasedExporter
	} else {
		tracesExporter = observability.NoopExporter
	}
	// Init Prometheus telemetry
	shutdown, err := observability.Start(ctx, logger, &observability.Options{
		MetricsExporter: observability.PrometheusExporter,
		TracesExporter:  tracesExporter,
		ServiceName:     "rill-local",
		ServiceVersion:  opts.Ch.Version.String(),
	})
	if err != nil {
		return nil, err
	}

	// Get full path to project. In multi-tenant mode use a shared work dir for storage/cache,
	// since per-client paths are resolved later by EnsureInstanceForClient.
	var projectPath, dbDirPath string
	if opts.MultiTenant {
		// Use a stable per-user work dir, e.g. ~/.rill/multi-tenant
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		projectPath = filepath.Join(homeDir, ".rill", "multi-tenant")
		dbDirPath = filepath.Join(projectPath, DefaultDBDir)
		if err := os.MkdirAll(dbDirPath, os.ModePerm); err != nil {
			return nil, err
		}
	} else {
		var err error
		projectPath, err = filepath.Abs(opts.ProjectPath)
		if err != nil {
			return nil, err
		}
		dbDirPath = filepath.Join(projectPath, DefaultDBDir)
		err = os.MkdirAll(dbDirPath, os.ModePerm) // Create project dir and db dir if it doesn't exist
		if err != nil {
			return nil, err
		}

		// old behaviour when data was stored in a stage.db file in the project directory.
		// drop old file, remove this code after some time
		_, err = os.Stat(filepath.Join(projectPath, "stage.db"))
		if err == nil { // a old stage.db file exists
			_ = os.Remove(filepath.Join(projectPath, "stage.db"))
			_ = os.Remove(filepath.Join(projectPath, "stage.db.wal"))
			logger.Info("Dropping old stage.db file and rebuilding project")
		}
	}

	// Create a local runtime with an in-memory metastore
	metastoreConfig, err := structpb.NewStruct(map[string]any{"dsn": "file:rill?mode=memory&cache=shared"})
	if err != nil {
		return nil, err
	}
	systemConnectors := []*runtimev1.Connector{
		{
			Type:   "sqlite",
			Name:   "metastore",
			Config: metastoreConfig,
		},
	}

	// Sender for sending transactional emails.
	// We use a noop sender by default, but you can uncomment the SMTP sender to send emails from localhost for testing.
	sender := email.NewNoopSender()
	// Uncomment to send emails for testing:
	// err = godotenv.Load()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to load .env file: %w", err)
	// }
	// smtpPort, err := strconv.Atoi(os.Getenv("RILL_RUNTIME_EMAIL_SMTP_PORT"))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get SMTP port: %w", err)
	// }
	// sender, err := email.NewSMTPSender(&email.SMTPOptions{
	// 	SMTPHost:     os.Getenv("RILL_RUNTIME_EMAIL_SMTP_HOST"),
	// 	SMTPPort:     smtpPort,
	// 	SMTPUsername: os.Getenv("RILL_RUNTIME_EMAIL_SMTP_USERNAME"),
	// 	SMTPPassword: os.Getenv("RILL_RUNTIME_EMAIL_SMTP_PASSWORD"),
	// 	FromEmail:    os.Getenv("RILL_RUNTIME_EMAIL_SENDER_EMAIL"),
	// 	FromName:     os.Getenv("RILL_RUNTIME_EMAIL_SENDER_NAME"),
	// 	BCC:          os.Getenv("RILL_RUNTIME_EMAIL_BCC"),
	// })
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create email sender: %w", err)
	// }
	rtOpts := &runtime.Options{
		ConnectionCacheSize:          100,
		MetastoreConnector:           "metastore",
		QueryCacheSizeBytes:          int64(datasize.MB * 100),
		AllowHostAccess:              allowHostAccess(), // see allow_host_access_*.go
		SystemConnectors:             systemConnectors,
		SecurityEngineCacheSize:      1000,
		ControllerLogBufferCapacity:  10000,
		ControllerLogBufferSizeBytes: int64(datasize.MB * 16),
		Version:                      opts.Ch.Version,
	}
	st, err := storage.New(dbDirPath, nil)
	if err != nil {
		return nil, err
	}
	rt, err := runtime.New(ctx, rtOpts, logger, st, opts.Ch.Telemetry(ctx), email.New(sender))
	if err != nil {
		return nil, err
	}

	// Merge opts.Variables with some local overrides of the defaults in runtime/drivers.InstanceConfig.
	vars := map[string]string{
		"rill.download_limit_bytes": "0", // 0 means unlimited
		"rill.stage_changes":        "false",
		"rill.watch_repo":           "true", // Run a file watcher instead of requiring manual refreshes
	}
	for k, v := range opts.Variables {
		vars[k] = v
	}

	// Prepare connectors for the instance
	var connectors []*runtimev1.Connector

	// Reset tmp dir
	if opts.Reset {
		_ = os.RemoveAll(dbDirPath)
		err = os.MkdirAll(dbDirPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	// Add default OLAP connector
	olapConfig, err := structpb.NewStruct(map[string]any{
		"pool_size":   "4",
		"log_queries": strconv.FormatBool(opts.Debug),
	})
	if err != nil {
		return nil, err
	}
	olapConnector := &runtimev1.Connector{
		Type:   "duckdb",
		Name:   "duckdb",
		Config: olapConfig,
	}
	connectors = append(connectors, olapConnector)

	// The repo connector is the local project directory
	repoConfig, err := structpb.NewStruct(map[string]any{
		"dsn":                   projectPath,
		"admin_url_override":    opts.Ch.AdminURLOverride,
		"access_token_override": opts.Ch.AdminTokenOverride,
		"home_dir":              opts.Ch.HomeDir,
	})
	if err != nil {
		return nil, err
	}
	repoConnector := &runtimev1.Connector{
		Type:   "file",
		Name:   "repo",
		Config: repoConfig,
	}
	connectors = append(connectors, repoConnector)

	// The catalog connector is a SQLite database in the project directory's tmp folder
	catalogConfig, err := structpb.NewStruct(map[string]any{"dsn": fmt.Sprintf("file:%s?cache=shared", filepath.Join(dbDirPath, DefaultCatalogStore))})
	if err != nil {
		return nil, err
	}
	catalogConnector := &runtimev1.Connector{
		Type:   "sqlite",
		Name:   "catalog",
		Config: catalogConfig,
	}
	connectors = append(connectors, catalogConnector)

	// Use the admin service for AI
	aiConfig, err := structpb.NewStruct(map[string]any{
		"admin_url":    opts.Ch.AdminURL(),
		"access_token": opts.Ch.AdminToken(),
	})
	if err != nil {
		return nil, err
	}
	aiConnector := &runtimev1.Connector{
		Name:   "admin",
		Type:   "admin",
		Config: aiConfig,
	}
	connectors = append(connectors, aiConnector)

	// Print start status – need to do it before creating the instance, since doing so immediately starts the controller
	isInit := IsProjectInit(projectPath)
	if isInit {
		sugarLogger.Infof("Hydrating project '%s'", projectPath)
	}

	// Determine the frontend URL based on whether we're serving the UI
	var frontendURL string
	if opts.ServeUI {
		// In production: The runtime serves the UI
		frontendURL = opts.LocalURL // e.g., "http://localhost:9009"
	} else {
		// In development: Vite serves the frontend on a separate port (3001)
		frontendURL = "http://localhost:3001"
	}

	// Create instance with its repo set to the project directory.
	// In multi-tenant mode skip this — instances are created lazily per-user.
	var inst *drivers.Instance
	if !opts.MultiTenant {
		inst = &drivers.Instance{
			ID:                               DefaultInstanceID,
			Environment:                      opts.Environment,
			OLAPConnector:                    olapConnector.Name,
			RepoConnector:                    repoConnector.Name,
			AIConnector:                      aiConnector.Name,
			CatalogConnector:                 catalogConnector.Name,
			Connectors:                       connectors,
			Variables:                        vars,
			Annotations:                      map[string]string{},
			IgnoreInitialInvalidProjectError: !isInit, // See ProjectParser reconciler for details
			FrontendURL:                      frontendURL,
		}
		err = rt.CreateInstance(ctx, inst)
		if err != nil {
			return nil, err
		}
	}

	// Resolve the projects directory for multi-tenant mode (defaults to $BRATRAX_PROJECTS_DIR or ./generated).
	projectsDir := opts.ProjectsDir
	if projectsDir == "" {
		projectsDir = os.Getenv("BRATRAX_PROJECTS_DIR")
	}
	if projectsDir == "" {
		projectsDir = "./generated"
	}
	if abs, absErr := filepath.Abs(projectsDir); absErr == nil {
		projectsDir = abs
	}
	if opts.MultiTenant {
		sugarLogger.Infof("Multi-tenant mode: per-client projects loaded from %s", projectsDir)

		// Create a real empty "default" instance so the frontend has a valid runtime
		// target before the user logs in. Without this, requests to /v1/instances/default/...
		// return 404 which TanStack Query retries infinitely, crashing the browser.
		// Once the user logs in, the InstanceRouterMiddleware rewrites "default" to their
		// real per-client instance ID.
		defaultOlapConfig, _ := structpb.NewStruct(map[string]any{"pool_size": "1"})
		defaultRepoConfig, _ := structpb.NewStruct(map[string]any{"dsn": projectPath})
		defaultCatalogConfig, _ := structpb.NewStruct(map[string]any{"dsn": fmt.Sprintf("file:%s?cache=shared", filepath.Join(dbDirPath, "default_catalog.db"))})
		defaultInst := &drivers.Instance{
			ID:               DefaultInstanceID,
			Environment:      opts.Environment,
			OLAPConnector:    "duckdb",
			RepoConnector:    "repo",
			CatalogConnector: "catalog",
			Connectors: []*runtimev1.Connector{
				{Type: "duckdb", Name: "duckdb", Config: defaultOlapConfig},
				{Type: "file", Name: "repo", Config: defaultRepoConfig},
				{Type: "sqlite", Name: "catalog", Config: defaultCatalogConfig},
			},
			Variables:                        vars,
			Annotations:                      map[string]string{},
			IgnoreInitialInvalidProjectError: true,
		}
		if err := rt.CreateInstance(ctx, defaultInst); err != nil {
			sugarLogger.Warnf("Failed to create default placeholder instance: %v", err)
		}
	}

	// Create app
	app := &App{
		Context:               ctx,
		Runtime:               rt,
		Instance:              inst,
		Logger:                sugarLogger,
		BaseLogger:            logger,
		Verbose:               opts.Verbose,
		Debug:                 opts.Debug,
		MultiTenant:           opts.MultiTenant,
		ProjectsDir:           projectsDir,
		environment:           opts.Environment,
		debugFlag:             opts.Debug,
		instances:             make(map[string]*drivers.Instance),
		ProjectPath:           projectPath,
		ch:                    opts.Ch,
		observabilityShutdown: shutdown,
		loggerCleanUp:         cleanupFn,
		pkceAuthenticators:    make(map[string]*pkce.Authenticator),
		localURL:              opts.LocalURL,
		allowedOrigins:        opts.AllowedOrigins,
	}

	// Collect and emit information about connectors at start time (skip in multi-tenant — no project)
	if !opts.MultiTenant {
		err = app.emitStartEvent(ctx)
		if err != nil {
			logger.Debug("failed to emit start event", zap.Error(err))
		}
	}

	return app, nil
}

func (a *App) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.observabilityShutdown(ctx)
	if err != nil {
		a.Logger.Error("Observability shutdown failed", zap.Error(err))
	}

	err = a.Runtime.Close()
	if err != nil {
		a.Logger.Error("Graceful shutdown failed", zap.Error(err))
	} else {
		a.Logger.Info("Rill shutdown gracefully")
	}

	a.loggerCleanUp()
	return nil
}

func (a *App) Serve(httpPort, grpcPort int, enableUI, openBrowser, readonly bool, userID, tlsCertPath, tlsKeyPath string) error {
	// Get analytics info
	installID, enabled, err := a.ch.DotRill.AnalyticsInfo()
	if err != nil {
		a.Logger.Warnf("error finding install ID: %v", err)
	}

	// Build local metadata. In multi-tenant mode there's no default instance — use a placeholder
	// instance ID; the bratrax instance-router middleware rewrites per-request.
	instanceID := DefaultInstanceID
	if a.Instance != nil {
		instanceID = a.Instance.ID
	}
	metadata := &localMetadata{
		InstanceID:       instanceID,
		GRPCPort:         grpcPort,
		InstallID:        installID,
		ProjectPath:      a.ProjectPath,
		UserID:           userID,
		Version:          a.ch.Version.Number,
		BuildCommit:      a.ch.Version.Commit,
		BuildTime:        a.ch.Version.Timestamp,
		IsDev:            a.ch.Version.IsDev(),
		AnalyticsEnabled: enabled,
		Readonly:         readonly,
	}

	// Create the local server handler
	localServer := &Server{
		logger:   a.BaseLogger,
		app:      a,
		metadata: metadata,
	}

	// Prepare errgroup and context with graceful shutdown
	gctx := graceful.WithCancelOnTerminate(a.Context)
	group, ctx := errgroup.WithContext(gctx)

	// Create server logger for the runtime
	runtimeServerLogger := a.BaseLogger
	if !a.Verbose {
		// It only logs error messages when !verbose to prevent lots of req/res info messages.
		runtimeServerLogger = a.BaseLogger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel))
	}

	// Create a runtime server
	opts := &runtimeserver.Options{
		HTTPPort:        httpPort,
		GRPCPort:        grpcPort,
		TLSCertPath:     tlsCertPath,
		TLSKeyPath:      tlsKeyPath,
		AllowedOrigins:  a.allowedOrigins,
		ServePrometheus: true,
	}
	runtimeServer, err := runtimeserver.NewServer(ctx, opts, a.Runtime, runtimeServerLogger, ratelimit.NewNoop(), a.ch.Telemetry(ctx), newLocalAdminService(a.ch, a.ProjectPath))
	if err != nil {
		return err
	}

	// if keypath and certpath are provided
	secure := tlsCertPath != "" && tlsKeyPath != ""

	// Start the local HTTP server. We build the runtime HTTP handler ourselves
	// so we can wrap it with the bratrax InstanceRouterMiddleware (multi-tenant mode).
	group.Go(func() error {
		var bratraxHandlers *bratrax.Handlers

		// applyDemoAI swaps a client's own BYOK key for the platform key on the
		// shared demo workspace, and pins that instance to a cheaper model (the
		// driver default is Opus, and here we pay rather than the customer). Every
		// other client keeps its own key and the driver default.
		//
		// Both instance-creation paths — the browser instance router and the MCP
		// ensure-ready — funnel through this, so a demo instance is identical no
		// matter which one created it first. Reads bratraxHandlers lazily: it is
		// assigned below, but these closures only run per-request.
		applyDemoAI := func(clientDB, byokKey string) (key, model string) {
			if bratraxHandlers == nil || bratraxHandlers.Config == nil {
				return byokKey, ""
			}
			cfg := bratraxHandlers.Config
			if cfg.AnthropicAPIKey != "" && clientDB == cfg.DemoClientSlug {
				return cfg.AnthropicAPIKey, cfg.DemoUsersModel
			}
			return byokKey, ""
		}

		runtimeHandler, err := runtimeServer.HTTPHandler(ctx, func(mux *http.ServeMux) {
			// Inject local-only endpoints on the runtime server
			localServer.RegisterHandlers(mux, httpPort, secure, enableUI)
			// Register Bratrax proxy (non-fatal — most users won't have Flask running).
			// In multi-tenant mode, hand the MCP handler a way to register + warm a
			// client's instance before proxying: the MCP inner route
			// (/v1/instances/{id}/mcp) bypasses InstanceRouterMiddleware, so without
			// this a cold instance returns a hard 400 "no server available". Stays
			// nil in single-tenant mode (the handler then just proxies as before).
			var ensureReady bratrax.EnsureReadyFn
			if a.MultiTenant {
				ensureReadyTimeout := defaultMCPEnsureReadyTimeout
				if v := os.Getenv("BRATRAX_MCP_ENSURE_READY_TIMEOUT_SECONDS"); v != "" {
					if secs, perr := strconv.Atoi(v); perr == nil && secs > 0 {
						ensureReadyTimeout = time.Duration(secs) * time.Second
					}
				}
				ensureReady = func(rctx context.Context, clientDB, key string) error {
					// Register the instance (idempotent — returns immediately if already
					// cached), then wait, bounded, for its controller to be ready.
					key, model := applyDemoAI(clientDB, key)
					if _, ensErr := a.EnsureInstanceForClient(rctx, clientDB, key, model); ensErr != nil {
						return ensErr
					}
					wctx, cancel := context.WithTimeout(rctx, ensureReadyTimeout)
					defer cancel()
					_, ctrlErr := a.Runtime.Controller(wctx, clientDB)
					return ctrlErr
				}
			}
			h, err := bratrax.RegisterHandlers(mux, a.BaseLogger, ensureReady)
			if err != nil {
				a.Logger.Warnf("bratrax proxy not registered: %v", err)
				return
			}
			bratraxHandlers = h

			// Internal endpoints — Flask uses these to notify the runtime when
			// per-client config changes (e.g. BYOK Anthropic key updates).
			// Auth via BRATRAX_INTERNAL_SECRET shared secret.
			if a.MultiTenant {
				bratrax.RegisterInternalHandlers(mux, func(clientDB string) error {
					return a.RefreshInstanceForClient(ctx, clientDB)
				}, a.BaseLogger)
			}
		}, enableUI)
		if err != nil {
			return err
		}

		// In multi-tenant mode, wrap the entire runtime handler with the instance router.
		// It rewrites /v1/instances/default/... → /v1/instances/{client.ClickhouseDB}/...
		// based on the bratrax_auth JWT cookie.
		serveHandler := runtimeHandler
		if a.MultiTenant && bratraxHandlers != nil {
			cs := bratraxHandlers.ClientStore
			ensure := func(clientDB string) (string, error) {
				// Look up the per-client Anthropic key (BYOK). Empty if unset —
				// instance still starts; chat fails at Open time and the frontend
				// shows an "add your key" CTA.
				key, lookupErr := cs.GetAnthropicKey(ctx, clientDB)
				if lookupErr != nil {
					a.BaseLogger.Warn("anthropic key lookup failed; continuing without",
						zap.String("clientDB", clientDB), zap.Error(lookupErr))
					key = ""
				}
				key, model := applyDemoAI(clientDB, key)
				return a.EnsureInstanceForClient(ctx, clientDB, key, model)
			}
			demoAI := bratrax.NewDemoAIQuota(bratraxHandlers.PromptStore, bratraxHandlers.Config)
			serveHandler = bratrax.InstanceRouterMiddleware(runtimeHandler, bratraxHandlers.AuthMapper, ensure, demoAI, a.BaseLogger)
			a.Logger.Info("Multi-tenant: instance router middleware installed")
		}

		return graceful.ServeHTTP(ctx, serveHandler, graceful.ServeOptions{
			Port:     httpPort,
			GRPCPort: grpcPort,
			CertPath: tlsCertPath,
			KeyPath:  tlsKeyPath,
			Logger:   a.BaseLogger,
		})
	})

	// Start debug server on port 6060
	if a.Debug {
		group.Go(func() error { return debugserver.ServeHTTP(ctx, 6060) })
	}

	// Open the browser when health check succeeds
	go a.PollServer(ctx, httpPort, enableUI && openBrowser, secure)

	// Run the server
	err = group.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("server crashed: %w", err)
	}

	return nil
}

func (a *App) PollServer(ctx context.Context, httpPort int, openOnHealthy, secure bool) {
	client := &http.Client{Timeout: time.Second}

	scheme := "http"
	if secure {
		scheme = "https"
		client.Transport = &http.Transport{
			// nolint:gosec // this is a health check against localhost, so it's safe to ignore the cert
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	uri := fmt.Sprintf("%s://localhost:%d", scheme, httpPort)

	for {
		// Wait a bit before (re)trying.
		//
		// We sleep before the first health check as a slightly hacky way to protect against the situation where
		// another Rill server is already running, which will pass the health check as a false positive.
		// By sleeping first, the ctx is in practice sure to have been cancelled with a "port taken" error at that point.
		select {
		case <-time.After(250 * time.Millisecond):
		case <-ctx.Done():
			return
		}

		// Check if server is up
		resp, err := client.Get(uri + "/v1/ping")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < http.StatusInternalServerError {
				break
			}
		}
	}

	// Health check succeeded
	a.Logger.Infof("Serving Rill on: %s", uri)
	if openOnHealthy {
		// Check for cancellation again to be safe
		if ctx.Err() != nil {
			return
		}

		// Open the browser
		err := browser.Open(uri)
		if err != nil {
			a.Logger.Debugf("could not open browser: %v", err)
		}
	}
}

// emitStartEvent sends a telemetry event with information about the project' state.
// It is not a blocking operation (events are flushed in the background).
func (a *App) emitStartEvent(ctx context.Context) error {
	repo, instanceID, err := cmdutil.RepoForProjectPath(a.ProjectPath)
	if err != nil {
		return err
	}

	p, err := parser.Parse(ctx, repo, instanceID, a.Instance.Environment, a.Instance.OLAPConnector)
	if err != nil {
		return err
	}

	connectors := p.AnalyzeConnectors(ctx)
	for _, c := range connectors {
		if c.Err != nil {
			return err
		}
	}

	var connectorNames []string
	for _, connector := range connectors {
		connectorNames = append(connectorNames, connector.Name)
	}

	a.ch.Telemetry(ctx).RecordBehavioralLegacy(activity.BehavioralEventAppStart, attribute.StringSlice("connectors", connectorNames), attribute.String("olap_connector", a.Instance.OLAPConnector))

	return nil
}

// IsProjectInit checks if the project is initialized by checking if rill.yaml exists in the project directory.
// It doesn't use any runtime functions since we need the ability to check this before creating the instance.
func IsProjectInit(projectPath string) bool {
	rillYAML := filepath.Join(projectPath, "rill.yaml")
	if _, err := os.Stat(rillYAML); err != nil {
		return false
	}
	return true
}

// EnsureInstanceForClient lazily creates a Rill instance for the given clientDB.
// The instance ID equals clientDB. The project is loaded from {ProjectsDir}/{clientDB}/rill.
// Safe to call concurrently. Returns the instance ID once ready.
// Returns bratrax.ErrProjectNotProvisioned if the project directory is missing.
//
// anthropicAPIKey is the per-client BYOK key for Claude chat. If empty, the
// instance is still created but the claude AI connector will fail at Open time
// (driver requires api_key); the frontend pre-checks /settings/ai and renders
// an "add your key" CTA so users don't hit that error path.
//
// anthropicModel overrides the claude driver's default model. Empty leaves the
// driver default in place, which is what every BYOK client gets; only the demo
// workspace sets it (to a cheaper model, since the platform pays for it).
func (a *App) EnsureInstanceForClient(ctx context.Context, clientDB, anthropicAPIKey, anthropicModel string) (string, error) {
	if !a.MultiTenant {
		// In single-tenant mode every request goes to the default instance.
		return DefaultInstanceID, nil
	}
	if clientDB == "" {
		return "", fmt.Errorf("EnsureInstanceForClient: empty clientDB")
	}

	a.instMu.Lock()
	defer a.instMu.Unlock()

	if _, ok := a.instances[clientDB]; ok {
		return clientDB, nil
	}

	// Verify the project dir exists on disk before touching the runtime.
	projectPath := filepath.Join(a.ProjectsDir, clientDB, "rill")
	if !IsProjectInit(projectPath) {
		return "", fmt.Errorf("%w: %s", bratrax.ErrProjectNotProvisioned, projectPath)
	}

	// Per-client tmp dir for DuckDB cache + catalog SQLite
	dbDirPath := filepath.Join(projectPath, DefaultDBDir)
	if err := os.MkdirAll(dbDirPath, os.ModePerm); err != nil {
		return "", err
	}

	// OLAP (DuckDB) connector
	olapConfig, err := structpb.NewStruct(map[string]any{
		// pool_size = max concurrent DuckDB connections this Rill instance
		// uses for reads + writes. Walked from 4 → 16 → 32.
		// At 16, isolated reconciles still finished but concurrent multi-instance
		// reconciles (9 active clients) showed per-instance RUNNING=19 — i.e. the
		// runtime wanted more slots than the pool offered, so 3 were queued.
		// Bumping to 32 to give the runtime headroom under multi-instance load.
		// DuckDB still serializes mutating ops at the page level — this only
		// helps when the actual contention is "pool-slot wait", not "file lock
		// wait." Watch CTAS write phase if regressions appear.
		"pool_size":   "32",
		"log_queries": strconv.FormatBool(a.debugFlag),
	})
	if err != nil {
		return "", err
	}
	olapConnector := &runtimev1.Connector{Type: "duckdb", Name: "duckdb", Config: olapConfig}

	// Repo connector pointing at the per-client project dir
	repoConfig, err := structpb.NewStruct(map[string]any{
		"dsn":                   projectPath,
		"admin_url_override":    a.ch.AdminURLOverride,
		"access_token_override": a.ch.AdminTokenOverride,
		"home_dir":              a.ch.HomeDir,
	})
	if err != nil {
		return "", err
	}
	repoConnector := &runtimev1.Connector{Type: "file", Name: "repo", Config: repoConfig}

	// Catalog SQLite per client
	catalogConfig, err := structpb.NewStruct(map[string]any{
		"dsn": fmt.Sprintf("file:%s?cache=shared", filepath.Join(dbDirPath, DefaultCatalogStore)),
	})
	if err != nil {
		return "", err
	}
	catalogConnector := &runtimev1.Connector{Type: "sqlite", Name: "catalog", Config: catalogConfig}

	// Per-client Claude AI connector — BYOK. The client's Anthropic API key is
	// looked up from rill_clients.anthropic_api_key by the caller and passed in.
	// If empty, we still register the connector so the rest of the instance starts
	// cleanly; the Claude driver will refuse Open at chat time (the frontend
	// pre-checks GET /settings/ai and shows an "add your key" CTA before then).
	aiProps := map[string]any{"api_key": anthropicAPIKey}
	if anthropicModel != "" {
		// Model overridden (demo workspace). Send no `temperature`: the newer
		// models reject it outright with 400 "`temperature` is deprecated for
		// this model", which would 400 every prompt.
		aiProps["model"] = anthropicModel
	} else {
		// Default model. Pin the temperature the claude driver used to apply as
		// its own default, so clients on the default model keep answering exactly
		// as they did before that default was removed from the driver.
		aiProps["temperature"] = defaultClaudeTemperature
	}
	aiConfig, err := structpb.NewStruct(aiProps)
	if err != nil {
		return "", err
	}
	aiConnector := &runtimev1.Connector{Type: "claude", Name: "claude", Config: aiConfig}

	connectors := []*runtimev1.Connector{olapConnector, repoConnector, catalogConnector, aiConnector}
	vars := map[string]string{
		"rill.download_limit_bytes": "0",
		// Stage source/model refreshes through a temp table + atomic rename so
		// dashboards keep serving old data until the new table is fully written.
		// Without this, every hourly cron tick DROPs the live table before
		// rebuilding it, which blanks any dashboard reading from it for the
		// duration of the CTAS.
		"rill.stage_changes": "true",
		"rill.watch_repo":    "true",
	}

	inst := &drivers.Instance{
		ID:                               clientDB,
		Environment:                      a.environment,
		OLAPConnector:                    olapConnector.Name,
		RepoConnector:                    repoConnector.Name,
		AIConnector:                      aiConnector.Name,
		CatalogConnector:                 catalogConnector.Name,
		Connectors:                       connectors,
		Variables:                        vars,
		Annotations:                      map[string]string{},
		IgnoreInitialInvalidProjectError: false,
		FrontendURL:                      a.localURL,
	}
	if err := a.Runtime.CreateInstance(ctx, inst); err != nil {
		return "", fmt.Errorf("create instance for client %q: %w", clientDB, err)
	}
	a.instances[clientDB] = inst
	a.Logger.Infof("Multi-tenant: created Rill instance %q from %s", clientDB, projectPath)
	return clientDB, nil
}

// RefreshInstanceForClient evicts the cached instance for clientDB and tears it
// down on the runtime. The next request that hits InstanceRouterMiddleware will
// trigger EnsureInstanceForClient again, which re-reads per-client config (e.g.
// the BYOK Anthropic key) from rill_clients. Used by Flask after /settings/ai
// updates so the new key takes effect without restarting the rill process.
//
// No-op (returns nil) in single-tenant mode or if the instance isn't cached.
func (a *App) RefreshInstanceForClient(ctx context.Context, clientDB string) error {
	if !a.MultiTenant {
		return nil
	}
	if clientDB == "" {
		return fmt.Errorf("RefreshInstanceForClient: empty clientDB")
	}

	a.instMu.Lock()
	_, cached := a.instances[clientDB]
	delete(a.instances, clientDB)
	a.instMu.Unlock()

	if !cached {
		// Nothing to tear down — next ensure will create fresh from current config.
		return nil
	}

	if err := a.Runtime.DeleteInstance(ctx, clientDB); err != nil {
		// Re-add to the cache so we don't end up in a half-evicted state on the
		// in-memory map while the runtime still holds the instance.
		a.Logger.Warnf("RefreshInstanceForClient: DeleteInstance failed for %q: %v", clientDB, err)
		return fmt.Errorf("delete instance %q: %w", clientDB, err)
	}
	a.Logger.Infof("Multi-tenant: evicted Rill instance %q (will be recreated on next request)", clientDB)
	return nil
}
