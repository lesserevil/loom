package loom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jordanhubbard/loom/internal/actions"
	"github.com/jordanhubbard/loom/internal/activity"
	"github.com/jordanhubbard/loom/internal/agent"
	"github.com/jordanhubbard/loom/internal/analytics"
	"github.com/jordanhubbard/loom/internal/beads"
	"github.com/jordanhubbard/loom/internal/collaboration"
	"github.com/jordanhubbard/loom/internal/comments"
	"github.com/jordanhubbard/loom/internal/consensus"
	"github.com/jordanhubbard/loom/internal/containers"
	"github.com/jordanhubbard/loom/internal/database"
	"github.com/jordanhubbard/loom/internal/decision"
	"github.com/jordanhubbard/loom/internal/dispatch"
	"github.com/jordanhubbard/loom/internal/eventbus"
	"github.com/jordanhubbard/loom/internal/executor"
	"github.com/jordanhubbard/loom/internal/files"
	"github.com/jordanhubbard/loom/internal/gitops"
	"github.com/jordanhubbard/loom/internal/keymanager"
	"github.com/jordanhubbard/loom/internal/logging"
	"github.com/jordanhubbard/loom/internal/memory"
	"github.com/jordanhubbard/loom/internal/messagebus"
	"github.com/jordanhubbard/loom/internal/meetings"
	"github.com/jordanhubbard/loom/internal/metrics"
	"github.com/jordanhubbard/loom/internal/modelcatalog"
	internalmodels "github.com/jordanhubbard/loom/internal/models"
	"github.com/jordanhubbard/loom/internal/motivation"
	"github.com/jordanhubbard/loom/internal/notifications"
	"github.com/jordanhubbard/loom/internal/observability"
	"github.com/jordanhubbard/loom/internal/openclaw"
	"github.com/jordanhubbard/loom/internal/orchestrator"
	"github.com/jordanhubbard/loom/internal/orgchart"
	"github.com/jordanhubbard/loom/internal/patterns"
	"github.com/jordanhubbard/loom/internal/persona"
	"github.com/jordanhubbard/loom/internal/project"
	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/internal/ralph"
	"github.com/jordanhubbard/loom/internal/statusboard"
	"github.com/jordanhubbard/loom/internal/swarm"
	"github.com/jordanhubbard/loom/internal/taskexecutor"
	"github.com/jordanhubbard/loom/internal/workflow"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/jordanhubbard/loom/pkg/connectors"
	"github.com/jordanhubbard/loom/pkg/models"
)

const readinessCacheTTL = 2 * time.Minute

type projectReadinessState struct {
	ready     bool
	issues    []string
	checkedAt time.Time
}

// Loom is the main orchestrator
type Loom struct {
	config                *config.Config
	agentManager          *agent.WorkerManager
	actionRouter          *actions.Router
	projectManager        *project.Manager
	personaManager        *persona.Manager
	beadsManager          *beads.Manager
	decisionManager       *decision.Manager
	fileLockManager       *FileLockManager
	orgChartManager       *orgchart.Manager
	providerRegistry      *provider.Registry
	database              *database.Database
	dispatcher            *dispatch.Dispatcher
	eventBus              *eventbus.EventBus
	modelCatalog          *modelcatalog.Catalog
	gitopsManager         *gitops.Manager
	shellExecutor         *executor.ShellExecutor
	logManager            *logging.Manager
	activityManager       *activity.Manager
	notificationManager   *notifications.Manager
	commentsManager       *comments.Manager
	collaborationStore    *collaboration.ContextStore
	consensusManager      *consensus.DecisionManager
	meetingsManager       *meetings.Manager
	motivationRegistry    *motivation.Registry
	motivationEngine      *motivation.Engine
	idleDetector          *motivation.IdleDetector
	workflowEngine        *workflow.Engine
	patternManager        *patterns.Manager
	metrics               *metrics.Metrics
	keyManager            *keymanager.KeyManager
	doltCoordinator       *beads.DoltCoordinator
	openclawClient        *openclaw.Client
	openclawBridge        *openclaw.Bridge
	containerOrchestrator *containers.Orchestrator
	connectorManager      *connectors.Manager
	memoryManager         *memory.MemoryManager
	messageBus            interface{}
	bridge                *messagebus.BridgedMessageBus
	pdaOrchestrator       *orchestrator.PDAOrchestrator
	swarmManager          *swarm.Manager
	swarmFederation       *swarm.Federation
	taskExecutor          *taskexecutor.Executor
	statusBoard           *statusboard.Board
	readinessMu           sync.Mutex
	readinessCache        map[string]projectReadinessState
	readinessFailures     map[string]time.Time
	shutdownOnce          sync.Once
	startedAt             time.Time
}

func New(cfg *config.Config) (*Loom, error) {
	personaPath := cfg.Agents.DefaultPersonaPath
	if personaPath == "" {
		personaPath = "./personas"
	}

	providerRegistry := provider.NewRegistry()

	// Initialize NATS message bus if configured
	var messageBus interface{}
	natsURL := os.Getenv("NATS_URL")
	if natsURL != "" {
		mbCfg := messagebus.Config{
			URL:        natsURL,
			StreamName: "LOOM",
			Timeout:    10 * time.Second,
		}
		mb, err := messagebus.NewNatsMessageBus(mbCfg)
		if err != nil {
			log.Printf("Warning: failed to initialize NATS message bus: %v", err)
			// Don't fail startup if NATS is unavailable - allow graceful degradation
		} else {
			messageBus = mb
			log.Printf("Initialized NATS message bus at %s", natsURL)
		}
	}

	// Initialize in-memory event bus.
	eb := eventbus.NewEventBus()

	// Bridge the in-memory EventBus to NATS for cross-container communication.
	var bridge *messagebus.BridgedMessageBus
	if messageBus != nil {
		if mb, ok := messageBus.(*messagebus.NatsMessageBus); ok {
			hostname, _ := os.Hostname()
			bridge = messagebus.NewBridgedMessageBus(mb, eb, "loom-control-"+hostname)
		}
	}

	// Initialize PostgreSQL database.
	// Config DSN takes priority; otherwise fall back to environment variables (POSTGRES_HOST, etc.)
	// An empty database Type means "no database" (skip initialization).
	var db *database.Database
	if cfg.Database.DSN != "" {
		var err error
		db, err = database.NewPostgres(cfg.Database.DSN)
		if err != nil {
			log.Printf("Warning: failed to initialize postgres: %v (running without persistence)", err)
		}
		log.Printf("Initialized postgres database from config DSN")
	} else if cfg.Database.Type != "" {
		var err error
		db, err = database.NewFromEnv()
		if err != nil {
			log.Printf("Warning: failed to initialize database: %v (running without persistence)", err)
		} else {
			log.Printf("Initialized postgres database from environment")
		}
	}

	// Initialize model catalog from config or use defaults.
	// Priority: 1) config.yaml preferred_models, 2) database override, 3) hardcoded defaults
	modelCatalog := modelcatalog.DefaultCatalog()
	if len(cfg.Models.PreferredModels) > 0 {
		// Convert config models to ModelSpec
		specs := make([]internalmodels.ModelSpec, 0, len(cfg.Models.PreferredModels))
		for _, pm := range cfg.Models.PreferredModels {
			spec := internalmodels.ModelSpec{
				Name:      pm.Name,
				Rank:      pm.Rank,
				MinVRAMGB: pm.MinVRAMGB,
			}
			// Map tier to interactivity
			switch pm.Tier {
			case "extended":
				spec.Interactivity = "slow"
			case "complex":
				spec.Interactivity = "medium"
			case "medium":
				spec.Interactivity = "medium"
			case "simple":
				spec.Interactivity = "fast"
			default:
				spec.Interactivity = "medium"
			}
			specs = append(specs, spec)
		}
		modelCatalog.Replace(specs)
		log.Printf("[ModelCatalog] Loaded %d preferred models from config.yaml", len(specs))
	}
	// Database can override config (for runtime updates via API)
	if db != nil {
		if raw, ok, err := db.GetConfigValue(modelCatalogKey); err == nil && ok {
			var specs []internalmodels.ModelSpec
			if err := json.Unmarshal([]byte(raw), &specs); err == nil && len(specs) > 0 {
				modelCatalog.Replace(specs)
				log.Printf("[ModelCatalog] Overrode with %d models from database", len(specs))
			}
		}
	}

	// Initialize gitops manager for project repository management.
	// baseWorkDir is where project repos are cloned to.
	// projectKeyDir is where SSH keys are stored (separate from clones to prevent
	// git stash/clean from destroying them).
	projectKeyDir := cfg.Git.ProjectKeyDir
	if projectKeyDir == "" {
		projectKeyDir = "/app/data/projects"
	}
	sshKeyDir := filepath.Join(filepath.Dir(projectKeyDir), "keys")
	gitopsMgr, err := gitops.NewManager(projectKeyDir, sshKeyDir, db, nil)
	if err != nil {
		log.Printf("Warning: failed to initialize gitops manager: %v", err)
	}
	gitopsMgr.SetSelfProjectID(cfg.GetSelfProjectID())

	// All projects are cloned consistently - no special workdir handling

	agentMgr := agent.NewWorkerManager(cfg.Agents.MaxConcurrent, providerRegistry, eb)
	if db != nil {
		agentMgr.SetAgentPersister(db)
		// Enable conversation context support for multi-turn conversations
		// Deprecated: WorkerPool is deprecated in favor of taskexecutor workers.
		// agentMgr.GetWorkerPool().SetDatabase(db)
	}

	// Initialize shell executor if database is available
	var shellExec *executor.ShellExecutor
	if db != nil {
		shellExec = executor.NewShellExecutor(db.DB())
	}
	var logMgr *logging.Manager
	if db != nil {
		logMgr = logging.NewManager(db.DB())
		logMgr.InstallLogInterceptor()
	}

	// Initialize motivation system
	motivationRegistry := motivation.NewRegistry(motivation.DefaultConfig())
	idleDetector := motivation.NewIdleDetector(motivation.DefaultIdleConfig())

	// Initialize workflow engine (if database is available)
	var workflowEngine *workflow.Engine
	if db != nil {
		beadsMgr := beads.NewManager(cfg.Beads.BDPath)
		workflowEngine = workflow.NewEngine(db, beadsMgr)
	}

	// Initialize activity, notification, and comments managers
	var activityMgr *activity.Manager
	var notificationMgr *notifications.Manager
	var commentsMgr *comments.Manager
	if db != nil {
		activityMgr = activity.NewManager(db, eb)
		notificationMgr = notifications.NewManager(db, activityMgr)
		commentsMgr = comments.NewManager(db, notificationMgr, eb)
	}

	// Initialize meetings manager
	var meetingsMgr *meetings.Manager
	if db != nil {
		meetingsMgr = meetings.NewManager()
	}
	// Initialize pattern manager and analytics logger if database is available
	var patternMgr *patterns.Manager
	if db != nil {
		analyticsStorage, err := analytics.NewDatabaseStorage(db.DB())
		if err != nil {
			log.Printf("Warning: failed to initialize analytics storage: %v", err)
		} else if analyticsStorage != nil {
			patternMgr = patterns.NewManager(analyticsStorage, nil)
			// Wire analytics logger to WorkerManager so LLM completions are logged
			agentMgr.SetAnalyticsLogger(analytics.NewLogger(analyticsStorage, analytics.DefaultPrivacyConfig()))
		}
	}

	// Initialize Dolt coordinator for multi-reader/multi-writer bead management
	// DISABLED: Let bd CLI manage Dolt in embedded mode to avoid lock conflicts
	var doltCoord *beads.DoltCoordinator
	// if cfg.Beads.Backend == "dolt" {
	// 	masterProject := cfg.GetSelfProjectID()
	// 	if len(cfg.Projects) > 0 {
	// 		masterProject = cfg.Projects[0].ID
	// 	}
	// 	doltCoord = beads.NewDoltCoordinator(masterProject, cfg.Beads.BDPath, 3307)
	// }

	// Initialize OpenClaw messaging gateway client and bridge (nil when disabled).
	ocClient := openclaw.NewClient(&cfg.OpenClaw)
	ocBridge := openclaw.NewBridge(ocClient, eb, &cfg.OpenClaw)

	// Initialize container orchestrator for per-project containers
	// Control plane URL for project agents to communicate back
	// Use container name "loom" as hostname (Docker network DNS resolution)
	controlPlaneURL := "http://loom:8081" // Port 8081 is the internal port
	if host := os.Getenv("CONTROL_PLANE_HOST"); host != "" {
		controlPlaneURL = fmt.Sprintf("http://%s:8081", host)
	}
	containerOrch, err := containers.NewOrchestrator(projectKeyDir, controlPlaneURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize container orchestrator: %w", err)
	}

	// Initialize connector manager for external service integrations
	connectorsConfigPath := filepath.Join("/app/data", "connectors.yaml")
	connectorMgr := connectors.NewManager(connectorsConfigPath)
	if err := connectorMgr.LoadConfig(); err != nil {
		log.Printf("Warning: Failed to load connectors config: %v", err)
	}
	// Start health monitoring for all connectors
	connectorMgr.StartHealthMonitoring(30 * time.Second)

	beadsMgr := beads.NewManager(cfg.Beads.BDPath)
	beadsMgr.SetBackend(cfg.Beads.Backend)

	collaborationStore := collaboration.NewContextStore()
	consensusManager := consensus.NewDecisionManager()
	statusBoard := statusboard.NewBoard()

	// Create motivation engine (will be wired after arb is created)
	var motivationEngine *motivation.Engine

	arb := &Loom{
		config:                cfg,
		startedAt:             time.Now().UTC(),
		agentManager:          agentMgr,
		projectManager:        project.NewManager(),
		personaManager:        persona.NewManager(personaPath),
		beadsManager:          beadsMgr,
		decisionManager:       decision.NewManager(),
		fileLockManager:       NewFileLockManager(cfg.Agents.FileLockTimeout),
		orgChartManager:       orgchart.NewManager(),
		providerRegistry:      providerRegistry,
		database:              db,
		eventBus:              eb,
		modelCatalog:          modelCatalog,
		gitopsManager:         gitopsMgr,
		shellExecutor:         shellExec,
		logManager:            logMgr,
		activityManager:       activityMgr,
		notificationManager:   notificationMgr,
		commentsManager:       commentsMgr,
		collaborationStore:    collaborationStore,
		consensusManager:      consensusManager,
		meetingsManager:       meetingsMgr,
		motivationRegistry:    motivationRegistry,
		idleDetector:          idleDetector,
		motivationEngine:      motivationEngine,
		workflowEngine:        workflowEngine,
		patternManager:        patternMgr,
		metrics:               metrics.NewMetrics(),
		doltCoordinator:       doltCoord,
		openclawClient:        ocClient,
		openclawBridge:        ocBridge,
		containerOrchestrator: containerOrch,
		connectorManager:      connectorMgr,
		messageBus:            messageBus,
		bridge:                bridge,
		statusBoard:           statusBoard,
	}

	buildEnv := actions.NewBuildEnvManager(providerRegistry)
	if containerOrch != nil {
		buildEnv.SetOnReady(containerOrch.SnapshotAfterSetup)
	}

	actionRouter := &actions.Router{
		Beads:         arb,
		Closer:        arb,
		Escalator:     arb,
		Commands:      arb,
		Files:         files.NewManager(gitopsMgr),
		Git:           actions.NewProjectGitRouter(gitopsMgr),
		Logger:        arb,
		Workflow:      arb,
		Projects:      arb,
		ContainerOrch: actions.NewContainerOrchAdapter(containerOrch),
		BuildEnv:      buildEnv,
		BeadType:      "task",
		BeadReader:    arb,
		DefaultP0:     true,
		Board:         arb.statusBoard,
		Meetings:      arb.meetingsManager,
		Consulter:     arb,
		Voter:         arb,
	}
	arb.actionRouter = actionRouter
	motivationEngine = motivation.NewEngine(motivationRegistry, arb, arb)
	arb.motivationEngine = motivationEngine
	agentMgr.SetActionRouter(actionRouter)

	// Enable multi-turn action loop
	agentMgr.SetActionLoopEnabled(true)
	agentMgr.SetMaxLoopIterations(100) // Increased to 100 to allow full development cycle (explore + plan + edit + build + test + commit)
	if db != nil {
		agentMgr.SetDatabase(db)
		lessonsProvider := dispatch.NewLessonsProvider(db)
		if lessonsProvider != nil {
			agentMgr.SetLessonsProvider(lessonsProvider)
		}
		arb.memoryManager = memory.NewMemoryManager(db)
	}

	arb.dispatcher = dispatch.NewDispatcher(arb.beadsManager, arb.projectManager, arb.agentManager, arb.providerRegistry, eb)
	arb.readinessCache = make(map[string]projectReadinessState)
	arb.readinessFailures = make(map[string]time.Time)
	arb.dispatcher.SetReadinessCheck(arb.CheckProjectReadiness)
arb.dispatcher.SetReadinessMode(dispatch.ReadinessMode(cfg.Readiness.Mode))
	arb.dispatcher.SetMaxDispatchHops(cfg.Dispatch.MaxHops)
	arb.dispatcher.SetEscalator(arb)
	// Enable conversation context support for multi-turn conversations
	if db != nil {
		arb.dispatcher.SetDatabase(db)
	}
	// Enable NATS message bus for async agent communication
	if messageBus != nil {
		if mb, ok := messageBus.(*messagebus.NatsMessageBus); ok {
			arb.dispatcher.SetMessageBus(mb)
			// Also configure container orchestrator with message bus
			if containerOrch != nil {
				containerOrch.SetMessageBus(mb)
			}
		}
	}

	// Wire git worktree manager for parallel agent isolation
	worktreeManager := gitops.NewGitWorktreeManager(projectKeyDir)
	arb.dispatcher.SetWorktreeManager(worktreeManager)

	// Wire container orchestrator for per-project isolation
	if containerOrch != nil {
		arb.dispatcher.SetContainerOrchestrator(containerOrch)
		if shellExec != nil {
			shellExec.SetContainerOrchestrator(containerOrch, arb.projectManager)
			shellExec.SetEnvReadyHook(func(ctx context.Context, projectID string, agent *containers.ProjectAgentClient) {
				if actionRouter.BuildEnv != nil {
					if err := actionRouter.BuildEnv.EnsureReady(ctx, projectID, agent); err != nil {
						log.Printf("[ShellExecutor] env init for %s failed (non-fatal): %v", projectID, err)
					}
				}
			})
		}
	}

	// Setup provider metrics tracking
	arb.setupProviderMetrics()

	return arb, nil
}
func (a *Loom) setupProviderMetrics() {
	if a.metrics == nil || a.providerRegistry == nil {
		return
	}

	a.providerRegistry.SetMetricsCallback(func(providerID string, success bool, latencyMs int64, totalTokens int64, errorCount int64) {
		if a.metrics != nil {
			a.metrics.RecordProviderRequest(providerID, "", success, latencyMs, totalTokens)
		}

		if a.database == nil {
			return
		}
		provider, err := a.database.GetProvider(providerID)
		if err != nil || provider == nil {
			return
		}
		if success {
			provider.RecordSuccess(latencyMs, totalTokens)
		} else {
			provider.RecordFailure()
		}
		_ = a.database.UpsertProvider(provider)

		if a.eventBus != nil {
			_ = a.eventBus.Publish(&eventbus.Event{
				Type: eventbus.EventTypeProviderUpdated,
				Data: map[string]interface{}{
					"provider_id":  providerID,
					"success":      success,
					"latency_ms":   latencyMs,
					"total_tokens": totalTokens,
				},
			})
		}
	})
}
func (a *Loom) Initialize(ctx context.Context) error {
	log.Printf("[Loom] DEBUG: Initialize started")
	// Prefer database-backed configuration when available.
	var projects []*models.Project
	if a.database != nil {
		storedProjects, err := a.database.ListProjects()
		if err != nil {
			return fmt.Errorf("failed to load projects: %w", err)
		}
		if len(storedProjects) > 0 {
			projects = storedProjects
			// Apply config overrides for fields not stored in the DB schema (e.g. UseContainer).
			cfgByID := make(map[string]struct{ UseContainer, UseWorktrees bool })
			for _, cp := range a.config.Projects {
				cfgByID[cp.ID] = struct{ UseContainer, UseWorktrees bool }{UseContainer: cp.UseContainer, UseWorktrees: cp.UseWorktrees}
			}
			for _, sp := range projects {
				if sp == nil {
					continue
				}
				if cfg, ok := cfgByID[sp.ID]; ok {
					sp.UseContainer = cfg.UseContainer
					sp.UseWorktrees = cfg.UseWorktrees
				}
			}
			known := map[string]struct{}{}
			for _, project := range storedProjects {
				if project == nil {
					continue
				}
				known[project.ID] = struct{}{}
			}
			for _, p := range a.config.Projects {
				if !p.IsSticky {
					continue
				}
				if _, ok := known[p.ID]; ok {
					continue
				}
				proj := &models.Project{
					ID:              p.ID,
					Name:            p.Name,
					GitRepo:         p.GitRepo,
					GitHubRepo:      p.GitHubRepo,
					Branch:          p.Branch,
					BeadsPath:       p.BeadsPath,
					GitAuthMethod:   models.GitAuthMethod(p.GitAuthMethod),
					GitStrategy:     normalizeGitStrategy(models.GitStrategy(p.GitStrategy)),
					GitCredentialID: p.GitCredentialID,
					IsPerpetual:     p.IsPerpetual,
					IsSticky:        p.IsSticky,
					UseContainer:    p.UseContainer,
					UseWorktrees:    p.UseWorktrees,
					Context:         p.Context,
					Status:          models.ProjectStatusOpen,
				}
				_ = a.database.UpsertProject(proj)
				projects = append(projects, proj)
			}
		} else {
			// Bootstrap from config.yaml into the configuration database.
			for _, p := range a.config.Projects {
				proj := &models.Project{
					ID:              p.ID,
					Name:            p.Name,
					GitRepo:         p.GitRepo,
					GitHubRepo:      p.GitHubRepo,
					Branch:          p.Branch,
					BeadsPath:       p.BeadsPath,
					GitAuthMethod:   models.GitAuthMethod(p.GitAuthMethod),
					GitStrategy:     normalizeGitStrategy(models.GitStrategy(p.GitStrategy)),
					GitCredentialID: p.GitCredentialID,
					IsPerpetual:     p.IsPerpetual,
					IsSticky:        p.IsSticky,
					UseWorktrees:    p.UseWorktrees,
					UseContainer:    p.UseContainer,
					Context:         p.Context,
					Status:          models.ProjectStatusOpen,
				}
				_ = a.database.UpsertProject(proj)
				projects = append(projects, proj)
			}
		}
	} else {
		for _, p := range a.config.Projects {
			projects = append(projects, &models.Project{
				ID:              p.ID,
				Name:            p.Name,
				GitRepo:         p.GitRepo,
				GitHubRepo:      p.GitHubRepo,
				Branch:          p.Branch,
				BeadsPath:       p.BeadsPath,
				GitAuthMethod:   models.GitAuthMethod(p.GitAuthMethod),
				GitStrategy:     normalizeGitStrategy(models.GitStrategy(p.GitStrategy)),
				GitCredentialID: p.GitCredentialID,
				IsPerpetual:     p.IsPerpetual,
				UseWorktrees:    p.UseWorktrees,
				IsSticky:        p.IsSticky,
				UseContainer:    p.UseContainer,
				Context:         p.Context,
				Status:          models.ProjectStatusOpen,
			})
		}
	}

	// Load projects into the in-memory project manager.
	var projectValues []models.Project
	for _, p := range projects {
		if p == nil {
			continue
		}
		copy := *p
		copy.BeadsPath = normalizeBeadsPath(copy.BeadsPath)
		copy.GitAuthMethod = normalizeGitAuthMethod(copy.GitRepo, copy.GitAuthMethod)
		projectValues = append(projectValues, copy)
	}
	if len(projectValues) == 0 && len(a.config.Projects) > 0 {
		for _, p := range a.config.Projects {
			projectValues = append(projectValues, models.Project{
				ID:              p.ID,
				Name:            p.Name,
				GitRepo:         p.GitRepo,
				GitHubRepo:      p.GitHubRepo,
				Branch:          p.Branch,
				BeadsPath:       normalizeBeadsPath(p.BeadsPath),
				GitAuthMethod:   normalizeGitAuthMethod(p.GitRepo, models.GitAuthMethod(p.GitAuthMethod)),
				GitStrategy:     normalizeGitStrategy(models.GitStrategy(p.GitStrategy)),
				GitCredentialID: p.GitCredentialID,
				UseWorktrees:    p.UseWorktrees,
				IsPerpetual:     p.IsPerpetual,
				IsSticky:        p.IsSticky,
				UseContainer:    p.UseContainer,
				Context:         p.Context,
				Status:          models.ProjectStatusOpen,
			})
		}
	}
	hasLoomProject := false
	for _, p := range projectValues {
		if p.ID == "loom" {
			hasLoomProject = true
			break
		}
	}
	if !hasLoomProject {
		projectValues = append(projectValues, models.Project{
			ID:            "loom",
			Name:          "Loom",
			GitRepo:       ".",
			Branch:        "main",
			BeadsPath:     normalizeBeadsPath(".beads"),
			GitAuthMethod: normalizeGitAuthMethod(".", ""),
			GitStrategy:   models.GitStrategyDirect,
			IsPerpetual:   true,
			IsSticky:      true,
			Context: map[string]string{
				"build_command": "make build",
				"test_command":  "make test",
				"lint_command":  "make lint",
			},
			Status: models.ProjectStatusOpen,
		})
	}
	if err := a.projectManager.LoadProjects(projectValues); err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}
	if a.database != nil {
		for i := range projectValues {
			p := projectValues[i]
			_ = a.database.UpsertProject(&p)
		}
	}

	// Load beads from registered projects.
	log.Printf("[Loom] DEBUG: Starting project loop, %d projects", len(projectValues))
	for i := range projectValues {
		p := &projectValues[i]
		if p.BeadsPath == "" {
			continue
		}

		// All projects are now treated consistently - clone from git
		// No special case for self project

		// Set default auth method if not specified
		if p.GitAuthMethod == "" {
			p.GitAuthMethod = models.GitAuthNone // Default to no auth for public repos
		}

		// For SSH-auth projects, ensure an SSH key exists
		if p.GitAuthMethod == models.GitAuthSSH {
			pubKey, err := a.gitopsManager.EnsureProjectSSHKey(p.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to ensure SSH key for %s: %v\n", p.ID, err)
			} else {
				fmt.Printf("Project %s SSH public key:\n%s\n", p.ID, pubKey)
			}
		}

		// Check if already cloned
		workDir := a.gitopsManager.GetProjectWorkDir(p.ID)
		p.WorkDir = workDir
		// Persist WorkDir so maintenance loop and dispatcher can find project files
		if mgdProject, _ := a.projectManager.GetProject(p.ID); mgdProject != nil {
			mgdProject.WorkDir = workDir
		}

		needsClone := false
		gitDir := filepath.Join(workDir, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			needsClone = true
		} else {
			// .git exists, but check if it's a valid clone (has commits)
			// An empty git-init repo with no commits means clone never succeeded
			checkCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
			checkCmd.Dir = workDir
			if out, err := checkCmd.CombinedOutput(); err != nil {
				outStr := strings.TrimSpace(string(out))
				if strings.Contains(outStr, "does not have any commits") || strings.Contains(outStr, "unknown revision") {
					fmt.Printf("Project %s has empty repo (prior clone failed), re-cloning...\n", p.ID)
					// Remove the broken repo so CloneProject can start fresh
					os.RemoveAll(workDir)
					needsClone = true
				}
			}
		}

		if needsClone {
			// Clone the repository
			fmt.Printf("Cloning project %s from %s...\n", p.ID, p.GitRepo)
			if err := a.gitopsManager.CloneProject(ctx, p); err != nil {
				errStr := err.Error()
				fmt.Fprintf(os.Stderr, "Warning: Failed to clone project %s: %v\n", p.ID, err)

				// If SSH auth failed, show the deploy key that needs to be registered
				if p.GitAuthMethod == models.GitAuthSSH && strings.Contains(errStr, "Permission denied") {
					if pubKey, keyErr := a.gitopsManager.EnsureProjectSSHKey(p.ID); keyErr == nil {
						fmt.Fprintf(os.Stderr, "\n"+
							"╔══════════════════════════════════════════════════════════════════╗\n"+
							"║  DEPLOY KEY NOT REGISTERED                                      ║\n"+
							"║                                                                  ║\n"+
							"║  Add this deploy key to your git remote:                         ║\n"+
							"║  %s\n"+
							"║                                                                  ║\n"+
							"║  For GitHub: Settings → Deploy Keys → Add deploy key             ║\n"+
							"║  Enable 'Allow write access' if agents need to push.             ║\n"+
							"╚══════════════════════════════════════════════════════════════════╝\n\n",
							pubKey)
					}
				}
				continue
			}
			fmt.Printf("Successfully cloned project %s\n", p.ID)
		} else {
			// Pull latest changes
			fmt.Printf("Pulling latest changes for project %s...\n", p.ID)
			if err := a.gitopsManager.PullProject(ctx, p); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to pull project %s: %v\n", p.ID, err)
				// Continue anyway with existing checkout
			} else {
				fmt.Printf("Successfully pulled project %s\n", p.ID)
			}
		}

		// Initialize beads database if needed.
		// For dolt backend, ensure bd is initialized with the correct prefix
		// so that bead creation doesn't fail with "database not initialized".
		{
			beadsDir := filepath.Join(workDir, p.BeadsPath)
			if _, err := os.Stat(beadsDir); err == nil {
				bdPath := a.config.Beads.BDPath
				if bdPath == "" {
					bdPath = "bd"
				}
				// Determine prefix for this project
				bdPrefix := p.BeadPrefix
				if bdPrefix == "" {
					bdPrefix = p.ID
				}
				initArgs := []string{"init", "--prefix", bdPrefix}
				if a.config.Beads.Backend != "dolt" {
					initArgs = append(initArgs, "--from-jsonl")
				}
				initCmd := exec.Command(bdPath, initArgs...)
				initCmd.Dir = workDir
				if out, err := initCmd.CombinedOutput(); err != nil {
					outStr := strings.TrimSpace(string(out))
					if !strings.Contains(outStr, "already initialized") {
						fmt.Fprintf(os.Stderr, "Warning: bd init failed for %s: %v (%s)\n", p.ID, err, outStr)
					}
				} else {
					fmt.Printf("Initialized beads database for project %s\n", p.ID)
				}
			}
		}

		// Update project in database with git metadata
		if a.database != nil {
			_ = a.database.UpsertProject(p)
		}

		// Setup git worktrees for project
		// Main branch at /app/data/projects/{id}/main
		// Beads branch at /app/data/projects/{id}/beads
		wtManager := gitops.NewGitWorktreeManager("/app/data/projects")
		beadsBranch := p.BeadsBranch
		if beadsBranch == "" {
			beadsBranch = a.config.Beads.BeadsBranch
		}
		if beadsBranch == "" {
			beadsBranch = "beads-sync" // Default
		}
		if err := wtManager.SetupBeadsWorktree(p.ID, p.Branch, beadsBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to setup beads worktree for %s: %v\n", p.ID, err)
		} else {
			log.Printf("[Loom] Setup beads worktree for project %s", p.ID)
		}

		// Load beads from the beads worktree
		beadsWorktree := wtManager.GetWorktreePath(p.ID, "beads")
		beadsPath := filepath.Join(beadsWorktree, p.BeadsPath)
		a.beadsManager.SetBeadsPath(beadsPath)
		// Track per-project beads path to avoid last-writer-wins across projects
		a.beadsManager.SetProjectBeadsPath(p.ID, beadsPath)
		// Configure git storage for this project
		a.beadsManager.SetGitStorage(p.ID, wtManager, beadsBranch, a.config.Beads.UseGitStorage, string(p.GitAuthMethod), p.GitRepo)
		// Load project prefix from config
		configPath := filepath.Join(beadsWorktree, p.BeadsPath)
		_ = a.beadsManager.LoadProjectPrefixFromConfig(p.ID, configPath)
		// Use project's BeadPrefix if set in the model
		if p.BeadPrefix != "" {
			a.beadsManager.SetProjectPrefix(p.ID, p.BeadPrefix)
		}
		// Load historical beads from main worktree first (baseline).
		// These may not yet be on the beads-sync branch.
		mainWorktree := wtManager.GetWorktreePath(p.ID, "main")
		mainBeadsPath := filepath.Join(mainWorktree, p.BeadsPath)
		if mainBeadsPath != beadsPath {
			_ = a.beadsManager.LoadBeadsFromFilesystem(p.ID, mainBeadsPath)
		}

		// Load from beads worktree (beads-sync branch) last - overwrites stale
		// main-worktree copies with authoritative beads-sync state.
		_ = a.beadsManager.LoadBeadsFromGit(ctx, p.ID, beadsPath)

		// Spawn isolated container for project if configured.
		// Run asynchronously so a slow Docker build/pull does not block startup.
		if p.UseContainer {
			projCopy := *p
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Fprintf(os.Stderr, "[Loom] PANIC in EnsureProjectContainer for %s: %v\n", projCopy.ID, r)
					}
				}()
				fmt.Fprintf(os.Stderr, "[Loom] Spawning isolated container for project %s (async)\n", projCopy.ID)
				bgCtx := context.Background()
				if err := a.containerOrchestrator.EnsureProjectContainer(bgCtx, &projCopy); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to start container for project %s: %v\n", projCopy.ID, err)
				} else {
					fmt.Fprintf(os.Stderr, "[Loom] Project %s container started successfully\n", projCopy.ID)
				}
				// Add a mechanism to signal completion or error
				// For example, using a channel to notify when done
			}()
		}

		// Start git-based federation (replaces Dolt)
		if a.config.Beads.Federation.Enabled && a.config.Beads.Federation.SyncMode == "git-native" {
			syncInterval := a.config.Beads.Federation.SyncInterval
			if syncInterval == 0 {
				syncInterval = 30 * time.Second // Default to 30 seconds
			}
			coordinator := beads.NewGitCoordinator(p.ID, wtManager, syncInterval)
			go coordinator.StartSyncLoop(ctx, a.beadsManager)
			log.Printf("[Loom] Started GitCoordinator for project %s", p.ID)
		}

	}

	// Load providers from database into the in-memory registry.
	if a.database != nil {
		providers, err := a.database.ListProviders()
		if err != nil {
			return fmt.Errorf("failed to load providers: %w", err)
		}
		if len(providers) == 0 && len(a.config.Providers) > 0 {
			for _, cfgProvider := range a.config.Providers {
				if !cfgProvider.Enabled {
					continue
				}
				providerID := cfgProvider.ID
				if providerID == "" && cfgProvider.Name != "" {
					providerID = strings.ReplaceAll(strings.ToLower(cfgProvider.Name), " ", "-")
				}
				if providerID == "" {
					log.Printf("Skipping provider seed without id or name: endpoint=%s", cfgProvider.Endpoint)
					continue
				}
				seed := &internalmodels.Provider{
					ID:          providerID,
					Name:        cfgProvider.Name,
					Type:        cfgProvider.Type,
					Endpoint:    cfgProvider.Endpoint,
					Model:       cfgProvider.Model,
					RequiresKey: cfgProvider.APIKey != "",
					Status:      "pending",
				}
				if _, regErr := a.RegisterProvider(ctx, seed); regErr != nil {
					log.Printf("Failed to seed provider %s: %v", providerID, regErr)
				}
			}
			providers, err = a.database.ListProviders()
			if err != nil {
				return fmt.Errorf("failed to reload providers: %w", err)
			}
		}

		// Auto-bootstrap or reconcile provider from LOOM_PROVIDER_URL env var.
		// If no providers exist, seed one. If the "tokenhub" provider exists but
		// its endpoint drifted (e.g. container network changed), update it so
		// workers don't keep hitting an unreachable address.
		if envURL := os.Getenv("LOOM_PROVIDER_URL"); envURL != "" {
			if len(providers) == 0 {
				log.Printf("[Loom] No providers configured — bootstrapping from LOOM_PROVIDER_URL: %s", envURL)
				envAPIKey := os.Getenv("LOOM_PROVIDER_API_KEY")
				seed := &internalmodels.Provider{
					ID:          "tokenhub",
					Name:        "TokenHub",
					Type:        "openai",
					Endpoint:    envURL,
					RequiresKey: envAPIKey != "",
					Status:      "pending",
				}
				if _, regErr := a.RegisterProvider(ctx, seed, envAPIKey); regErr != nil {
					log.Printf("[Loom] Failed to bootstrap provider from env: %v", regErr)
				}
				providers, _ = a.database.ListProviders()
			} else {
				for _, p := range providers {
					if p.ID == "tokenhub" && p.Endpoint != envURL {
						log.Printf("[Loom] Reconciling tokenhub endpoint: %s → %s", p.Endpoint, envURL)
						p.Endpoint = envURL
						if dbErr := a.database.UpsertProvider(p); dbErr != nil {
							log.Printf("[Loom] Failed to reconcile tokenhub endpoint: %v", dbErr)
						}
						break
					}
				}
			}
		}
		for _, p := range providers {
			selected := p.SelectedModel
			if selected == "" {
				selected = p.Model
			}
			if selected == "" {
				selected = p.ConfiguredModel
			}
			var apiKey string
			if p.KeyID != "" && a.keyManager != nil && a.keyManager.IsUnlocked() {
				apiKey, _ = a.keyManager.GetKey(p.KeyID)
			}
			if apiKey == "" {
				apiKey = p.APIKey // fall back to key stored directly in provider record
			}
			_ = a.providerRegistry.Upsert(&provider.ProviderConfig{
				ID:                     p.ID,
				Name:                   p.Name,
				Type:                   p.Type,
				Endpoint:               normalizeProviderEndpoint(p.Endpoint),
				APIKey:                 apiKey,
				Model:                  selected,
				ConfiguredModel:        p.ConfiguredModel,
				SelectedModel:          selected,
				Status:                 p.Status,
				LastHeartbeatAt:        p.LastHeartbeatAt,
				LastHeartbeatLatencyMs: p.LastHeartbeatLatencyMs,
			})
		}

		// Count providers ready for dispatch; re-probe any that aren't healthy.
		// checkProviderHealthAndActivate is normally called when a provider is first
		// registered. On restart, providers are loaded from DB via Upsert (no probe),
		// so we must re-probe them here to promote unhealthy/pending ones to active.
		healthyCount := 0
		for _, p := range providers {
			if p.Status == "healthy" || p.Status == "active" {
				healthyCount++
			} else {
				pID := p.ID
				go a.checkProviderHealthAndActivate(pID)
			}
		}
		if healthyCount > 0 {
			log.Printf("[Loom] %d providers already healthy, dispatch ready", healthyCount)
		} else {
			log.Printf("[Loom] No providers healthy yet — probing all providers now")
		}

		// Restore agents from database (best-effort).
		storedAgents, err := a.database.ListAgents()
		if err != nil {
			return fmt.Errorf("failed to load agents: %w", err)
		}
		for _, ag := range storedAgents {
			if ag == nil {
				continue
			}
			// Attach persona (required for the system prompt).
			persona, err := a.personaManager.LoadPersona(ag.PersonaName)
			if err != nil {
				continue
			}
			ag.Persona = persona
			// Ensure a provider exists.
			if ag.ProviderID == "" {
				providers := a.providerRegistry.ListActive()
				if len(providers) == 0 {
					continue
				}
				ag.ProviderID = providers[0].Config.ID
			}
			_, _ = a.agentManager.RestoreAgentWorker(ctx, ag)
			_ = a.projectManager.AddAgentToProject(ag.ProjectID, ag.ID)
		}
	}

	// Ensure all projects are persisted to the database before creating agents (to avoid FK constraint failures)
	if a.database != nil {
		log.Printf("Persisting %d project(s) to database before agent creation", len(projectValues))
		for i := range projectValues {
			p := &projectValues[i]
			if err := a.database.UpsertProject((*models.Project)(p)); err != nil {
				log.Printf("Warning: Failed to persist project %s: %v", p.ID, err)
			} else {
				log.Printf("Successfully persisted project %s to database", p.ID)
			}
		}
	}

	// Ensure default agents are assigned for each project.
	for _, p := range projectValues {
		if p.ID == "" {
			continue
		}
		_ = a.ensureDefaultAgents(ctx, p.ID)
	}

	// After restoring agents from DB, reset any that were left in "working" state.
	// They have no running goroutine after restart, so clearing their status allows
	// the dispatch loop to re-assign their beads on the first tick.
	if resetCount := a.agentManager.ResetStuckAgents(0); resetCount > 0 {
		log.Printf("[Loom] Reset %d agent(s) left in 'working' state from previous run", resetCount)
	}

	// Reset any beads left in_progress with ephemeral executor IDs from the previous
	// run. Agent status reset above only covers named agents; exec-* goroutine IDs
	// die silently on restart and must be cleaned up here so the task executor can
	// reclaim the work immediately.
	if zombieCount := a.resetZombieBeads(); zombieCount > 0 {
		log.Printf("[Loom] Reset %d zombie bead(s) left in 'in_progress' state from previous run", zombieCount)
	}

	// Attach healthy providers to any paused agents after creating default agents
	// Small delay to ensure agents are persisted to database
	time.Sleep(500 * time.Millisecond)
	healthyProviders := a.providerRegistry.ListActive()
	for _, provider := range healthyProviders {
		if provider != nil && provider.Config != nil {
			log.Printf("Attaching healthy provider %s to paused agents on startup", provider.Config.ID)
			a.attachProviderToPausedAgents(ctx, provider.Config.ID)
		}
	}

	// Start the Ralph Loop — a plain goroutine ticker that runs maintenance
	// every 10 seconds (resets stuck agents, auto-blocks looped beads, etc.).
	ralphActs := ralph.New(a.database, a.dispatcher, a.beadsManager, a.agentManager)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		beatCount := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				beatCount++
				if err := ralphActs.Beat(ctx, beatCount); err != nil {
					log.Printf("[Ralph] Beat %d failed: %v", beatCount, err)
				}
			}
		}
		// Add a mechanism to signal completion or error
		// For example, using a channel to notify when done
	}()

	// Kick-start work on all open beads across registered projects.
	a.kickstartOpenBeads(ctx)

	// Register default motivations for all agent roles
	if a.motivationRegistry != nil {
		if err := motivation.RegisterDefaults(a.motivationRegistry); err != nil {
			log.Printf("Warning: Failed to register default motivations: %v", err)
		} else {
			log.Printf("Registered %d default motivations", a.motivationRegistry.Count())
		}
	}

	// FIX #4: Ensure at least one project has beads for work to flow
	// If no beads exist across all projects, create a diagnostic bead
	hasBeads := false
	allProjects := a.projectManager.ListProjects()
	for _, proj := range allProjects {
		if proj == nil {
			continue
		}
		beads, _ := a.beadsManager.ListBeads(map[string]interface{}{"project_id": proj.ID})
		if len(beads) > 0 {
			hasBeads = true
			break
		}
	}

	// If no beads exist and we have at least one project, create a sample diagnostic bead
	if !hasBeads && len(allProjects) > 0 {
		proj := allProjects[0]
		log.Printf("[Loom] No beads found - creating sample diagnostic bead for project %s", proj.ID)

		bead, err := a.beadsManager.CreateBead(
			"System diagnostic check",
			`This is an automated diagnostic task to verify the Loom workflow is operational.

## Your Task

1. Run the project build command to verify the build system works
2. Run the project tests to verify the test system works
3. If both pass, use the 'done' action with reason "Diagnostic complete: build and tests pass"
4. If either fails, use the 'done' action with reason explaining what failed

This is a simple verification task. Do NOT search for bugs or make changes. Just verify build and test, then mark done.`,
			models.BeadPriorityP2,
			"task",
			proj.ID,
		)
		if err != nil {
			log.Printf("[Loom] Failed to create sample diagnostic bead: %v", err)
		} else {
			log.Printf("[Loom] Created sample diagnostic bead: %s", bead.ID)
		}
	} else if len(allProjects) == 0 {
		log.Printf("[Loom] Warning: No projects configured - no work can be dispatched")
	} else {
		log.Printf("[Loom] Found existing beads across projects - work flow should be operational")
	}

	// Load default workflows
	if a.database != nil && a.workflowEngine != nil {
		workflowsDir := "./workflows/defaults"
		if _, err := os.Stat(workflowsDir); err == nil {
			log.Printf("Loading default workflows from %s", workflowsDir)
			if err := workflow.InstallDefaultWorkflows(a.database, workflowsDir); err != nil {
				log.Printf("Warning: Failed to load default workflows: %v", err)
			} else {
				log.Printf("Successfully loaded default workflows")
			}
		} else {
			log.Printf("Default workflows directory not found: %s", workflowsDir)
		}

		// Set workflow engine in dispatcher for workflow-aware routing
		if a.dispatcher != nil {
			a.dispatcher.SetWorkflowEngine(a.workflowEngine)
			log.Printf("Workflow engine connected to dispatcher")
		}
	}

	// ── Multi-service pub/sub wiring ───────────────────────────────────
	// Start the NATS ↔ EventBus bridge so cross-container events flow.
	if a.bridge != nil {
		if err := a.bridge.Start(ctx); err != nil {
			log.Printf("[Loom] Warning: Failed to start NATS bridge: %v", err)
		}
	}

	// Apply UseNATSDispatch feature flag from config.
	if a.config.Dispatch.UseNATSDispatch && a.messageBus != nil {
		a.dispatcher.SetUseNATSDispatch(true)
		log.Printf("[Loom] NATS dispatch enabled – tasks will be routed to agent containers")
	}

	// Start PDA orchestrator if enabled.
	if a.config.PDA.Enabled && a.messageBus != nil {
		if mb, ok := a.messageBus.(*messagebus.NatsMessageBus); ok {
			var planner orchestrator.Planner
			if a.config.PDA.PlannerEndpoint != "" {
				planner = orchestrator.NewLLMPlanner(
					a.config.PDA.PlannerEndpoint,
					a.config.PDA.PlannerAPIKey,
					a.config.PDA.PlannerModel,
				)
			} else {
				planner = &orchestrator.StaticPlanner{}
			}
			adapter := orchestrator.NewBeadManagerAdapter(a.beadsManager)
			a.pdaOrchestrator = orchestrator.NewPDAOrchestrator(mb, planner, adapter, adapter)
			if err := a.pdaOrchestrator.Start(ctx); err != nil {
				log.Printf("[Loom] Warning: Failed to start PDA orchestrator: %v", err)
			}
		}
	}

	// Start swarm manager if enabled.
	if a.config.Swarm.Enabled && a.messageBus != nil {
		if mb, ok := a.messageBus.(*messagebus.NatsMessageBus); ok {
			hostname, _ := os.Hostname()
			a.swarmManager = swarm.NewManager(mb, "loom-control-plane", "control-plane")
			var projectIDs []string
			for _, p := range a.config.Projects {
				projectIDs = append(projectIDs, p.ID)
			}
			port := a.config.Server.HTTPPort
			endpoint := fmt.Sprintf("http://%s:%d", hostname, port)
			if err := a.swarmManager.Start(ctx, []string{"control-plane"}, projectIDs, endpoint); err != nil {
				log.Printf("[Loom] Warning: Failed to start swarm manager: %v", err)
			}
			// Wire swarm manager to dispatcher for dynamic service discovery routing.
			if a.dispatcher != nil {
				a.dispatcher.SetSwarmManager(a.swarmManager)
				if a.memoryManager != nil {
					a.dispatcher.SetMemoryManager(a.memoryManager)
				}
			}

			// Federation with peer NATS instances
			if len(a.config.Swarm.PeerNATSURLs) > 0 {
				a.swarmFederation = swarm.NewFederation(mb, swarm.FederationConfig{
					PeerNATSURLs: a.config.Swarm.PeerNATSURLs,
					GatewayName:  a.config.Swarm.GatewayName,
				})
				if err := a.swarmFederation.Start(ctx); err != nil {
					log.Printf("[Loom] Warning: Failed to start federation: %v", err)
				}
			}
		}
	}

	// Start motivation engine
	if a.motivationEngine != nil {
		if err := a.motivationEngine.Start(ctx); err != nil {
			log.Printf("[Loom] Warning: Failed to start motivation engine: %v", err)
		}
	}

	log.Printf("[Loom] DEBUG: Initialize completed successfully")
	return nil
}

// kickstartOpenBeads starts Temporal workflows for all open beads in registered projects.
func (a *Loom) Shutdown() {
	a.shutdownOnce.Do(func() {
		if a.agentManager != nil {
			a.agentManager.StopAll()
		}
		if a.connectorManager != nil {
			_ = a.connectorManager.Close()
		}
		if a.pdaOrchestrator != nil {
			a.pdaOrchestrator.Close()
		}
		if a.swarmFederation != nil {
			a.swarmFederation.Close()
		}
		if a.swarmManager != nil {
			a.swarmManager.Close()
		}
		if a.bridge != nil {
			a.bridge.Close()
		}
		if a.openclawBridge != nil {
			a.openclawBridge.Close()
		}
		if a.doltCoordinator != nil {
			a.doltCoordinator.Shutdown()
		}
		if a.eventBus != nil {
			a.eventBus.Close()
		}
		if a.messageBus != nil {
			if mb, ok := a.messageBus.(*messagebus.NatsMessageBus); ok {
				_ = mb.Close()
			}
		}
		if a.database != nil {
			_ = a.database.Close()
		}
	})
}
