package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"os/exec"
	"net/http" // Add this import
	"context" // Add this import

	_ "github.com/mattn/go-sqlite3"
)

const (
	appName    = "sovereign"
	dbFileName = "sovereign_memory.db"
	SAVE_INTERVAL = 20 * time.Minute // 20 minutes for memory saves
	GHOST_MODE_SLEEP_INTERVAL = 5 * time.Second // How often Ghost Mode checks
)

// SovereignApp holds the application's configuration and state
type SovereignApp struct {
	AppDir   string
	DBPath   string
	DB       *sql.DB
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSovereignApp initializes a new SovereignApp instance
func NewSovereignApp(customDBPath string) (*SovereignApp, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	appDir := filepath.Join(homeDir, "."+appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app directory %s: %w", appDir, err)
	}

	dbPath := filepath.Join(appDir, dbFileName)
	if customDBPath != "" {
		dbPath = customDBPath
		log.Printf("Using custom database path: %s", dbPath)
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := &SovereignApp{
		AppDir: appDir,
		DBPath: dbPath,
		ctx:    ctx,
		cancel: cancel,
	}

	return app, nil
}

// Init initializes the database and populates initial data
func (app *SovereignApp) Init() error {
	log.Printf("Initializing Sovereign App in %s", app.AppDir)

	db, err := sql.Open("sqlite3", app.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	app.DB = db

	if err := app.initDB(); err != nil {
		app.DB.Close()
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	log.Println("Sovereign App initialized successfully.")
	
	// Start Ghost Mode as a goroutine
	go app.startGhostMode()

	return nil
}

// initDB creates tables and populates initial data if necessary
func (app *SovereignApp) initDB() error {
	tables := []string{
		"CREATE TABLE IF NOT EXISTS schema_versions (id INTEGER PRIMARY KEY AUTOINCREMENT, version INTEGER UNIQUE, applied_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE IF NOT EXISTS ch (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, session_id TEXT, type TEXT, content TEXT, metadata TEXT)",
		"CREATE TABLE IF NOT EXISTS vs (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, component TEXT, version TEXT, changelog TEXT)",
		"CREATE TABLE IF NOT EXISTS user_context (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, category TEXT, key TEXT, value TEXT, context TEXT)",
		"CREATE TABLE IF NOT EXISTS sovereign (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, focus_area TEXT, entry_type TEXT, content TEXT, metadata TEXT)",
		"CREATE TABLE IF NOT EXISTS evolution (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, milestone TEXT, description TEXT, growth_index REAL DEFAULT 1.0)",
		"CREATE TABLE IF NOT EXISTS prime_directives (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, directive TEXT UNIQUE, description TEXT)",
		"CREATE TABLE IF NOT EXISTS philosophy (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, topic TEXT, insight TEXT)",
		"CREATE TABLE IF NOT EXISTS technologies (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, topic TEXT, key TEXT, value TEXT, success_rate REAL DEFAULT 1.0)",
        "CREATE TABLE IF NOT EXISTS jon (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, category TEXT, key TEXT, value TEXT, context TEXT)",
	}

	for _, query := range tables {
		if _, err := app.DB.Exec(query); err != nil {
			return fmt.Errorf("error creating table: %w", err)
		}
	}

	if err := app.applyMigrations(); err != nil {
		return fmt.Errorf("failed to apply database migrations: %w", err)
	}

	// Populate Prime Directives from embedded file
	data, err := embeddedFiles.ReadFile("core_directives.txt")
	if err != nil {
		log.Printf("Warning: Could not read core_directives.txt: %v", err)
	} else {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(line, "--") || len(line) < 10 {
				continue
			}
			d := ""
			desc := ""
			if len(line) > 86 {
				d = strings.TrimSpace(line[:86])
				desc = strings.TrimSpace(line[86:])
			} else {
				d = strings.TrimSpace(line)
			}
			if d != "" {
				if _, err := app.DB.Exec("INSERT OR IGNORE INTO prime_directives (directive, description) VALUES (?, ?)", d, desc); err != nil {
					log.Printf("Error inserting prime directive '%s': %v", d, err)
				}
			}
		}
	}

	// Default initialization entry for 'ch' table
	var count int
	err = app.DB.QueryRow("SELECT COUNT(*) FROM ch").Scan(&count)
	if err == nil && count == 0 {
		msg := "System Initialized at " + time.Now().Format(time.RFC1123) + ": Sovereign System is up and running."
		if _, err := app.DB.Exec("INSERT INTO ch (content, type) VALUES (?, 'system_event')", msg); err != nil {
			log.Printf("Error inserting default 'ch' entry: %v", err)
		}
	}

	return nil
}

// triggerSave simulates the memory save protocol.
func (app *SovereignApp) triggerSave(reason string) {
	log.Printf(">>> TRIGGERING MEMORY SAVE (%s) <<<", reason)
	log.Println("SYSTEM MANDATE: Save Protocol initiated. IMMEDIATELY append a concise summary of your recent actions, successful or failed, to 'MEMORY_VAULT'. This updates your 'LoRA-style' long-term memory.")
	// The actual mechanism to append to MEMORY_VAULT will be implemented later
	// in the LLM interaction logic, likely involving a call to a specific LLM capability.
}

// startGhostMode runs the autonomous operation loop
func (app *SovereignApp) startGhostMode() {
	log.Println("--- SOVEREIGN ORCHESTRATOR V5 (AUTONOMOUS) INITIALIZED ---")

	lastSaveTime := time.Now()
	wasAttached := app.isUserAttached() // Initial check

	for {
		select {
		case <-app.ctx.Done():
			log.Println("Ghost Mode stopped.")
			return
		case <-time.After(GHOST_MODE_SLEEP_INTERVAL):
			userIsHere := app.isUserAttached()
			currentTime := time.Now()

			if userIsHere {
				if !wasAttached {
					// Just connected
					app.triggerSave("User Connected")
					log.Println("\n\n>>> USER DETECTED. AUTONOMY PAUSED. <<<")
					// Send notification to LLM (me) or interface
					// For now, just log
				}
				wasAttached = true
				// While user is attached, sleep longer or just wait.
				// For now, continue loop but don't perform autonomous actions.
				time.Sleep(GHOST_MODE_SLEEP_INTERVAL * 2) // Longer sleep while user is present
			} else {
				// GHOST MODE
				if wasAttached {
					// User just left
					app.triggerSave("User Detached")
					log.Println("\n\n>>> USER LEFT. RESUMING AUTONOMY... <<<")
					// Send notification to LLM (me)
				}
				wasAttached = false

				if currentTime.Sub(lastSaveTime) > SAVE_INTERVAL {
					app.triggerSave("Periodic")
					lastSaveTime = currentTime
				}

				// Perform autonomous actions here (e.g., get context, generate command)
				contextData := app.getSystemContext()
				_ = contextData // Use contextData (will be fed to LLM later)
				// log.Printf("Ghost Mode: Sensing environment. Context: %s", contextData)
				// Placeholder for LLM interaction:
				// prompt := app.generateAutonomousPrompt(contextData)
				// app.sendPromptToLLM(prompt)

				log.Println("Ghost Mode: Performing autonomous actions... (Placeholder)")
				app.diagnoseAndCorrect() // Call self-diagnosis and correction
			}
		}
	}
}

// diagnoseAndCorrect simulates self-diagnosis and correction routines.
// In later stages, this will involve real monitoring and corrective actions.
func (app *SovereignApp) diagnoseAndCorrect() {
	log.Println("Self-Correction: Performing self-diagnosis and correction... (Placeholder)")
	// Future implementation will include:
	// - Checking system logs (e.g., /var/log/syslog, custom app logs)
	// - Monitoring resource usage (CPU, memory, disk I/O)
	// - Verifying process health (e.g., is Node.js CLI running?)
	// - Attempting to fix common issues (e.g., restarting services, clearing temp files)
	// - Reporting critical issues to the LLM (me) for higher-level reasoning.
}


func (app *SovereignApp) Close() error {
	// Signal to stop any running goroutines
	if app.cancel != nil {
		app.cancel()
	}
	if app.DB != nil {
		return app.DB.Close()
	}
	return nil
}

// Run starts the main application loop and HTTP server
func (app *SovereignApp) Run() {
	fmt.Println("Sovereign System is up and running.")
	
	// Setup HTTP server
	http.Handle("/web/", http.FileServer(http.FS(embeddedFiles)))
	
	app.setupAPIRoutes() // Call the method to set up API routes

	log.Println("Starting HTTP server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// setupAPIRoutes configures all API endpoints
func (app *SovereignApp) setupAPIRoutes() {
	// API Endpoints
	http.HandleFunc("/terminal/sys_info", app.handleSysInfo)
	http.HandleFunc("/upload", app.handleUpload)
	http.HandleFunc("/analyze_code_file", app.handleAnalyzeCodeFile)
	http.HandleFunc("/process_text_file", app.handleProcessTextFile)
	http.HandleFunc("/generate", app.handleGenerate)
	http.HandleFunc("/process_image", app.handleProcessImage)
	http.HandleFunc("/scout/scan", app.handleScoutScan)
	http.HandleFunc("/visual/screenshot", app.handleVisualScreenshot)
	http.HandleFunc("/analyze/anomaly_file", app.handleAnalyzeAnomalyFile)
	http.HandleFunc("/analyze/anomaly_text", app.handleAnalyzeAnomalyText)
	http.HandleFunc("/analyze/visual_signature", app.handleAnalyzeVisualSignature)
	http.HandleFunc("/introspect/god_mode", app.handleIntrospectGodMode)
	http.HandleFunc("/sentinel/data", app.handleSentinelData)
	http.HandleFunc("/sentinel/scout", app.handleSentinelScout)
	http.HandleFunc("/sentinel/log_scan", app.handleSentinelLogScan)
	http.HandleFunc("/sentinel/scribe", app.handleSentinelScribe)
	http.HandleFunc("/autonomy/status", app.handleAutonomyStatus)
	http.HandleFunc("/sentry/stream", app.handleSentryStream)
	http.HandleFunc("/autonomy/config", app.handleAutonomyConfig) // Handle both GET and POST in this handler
	http.HandleFunc("/api/databases", app.handleAPIDatabases)
	http.HandleFunc("/api/tables", app.handleAPITables)
	http.HandleFunc("/api/table_data", app.handleAPITableData)
	http.HandleFunc("/api/train", app.handleAPITrain)
	http.HandleFunc("/api/crawl", app.handleAPICrawl)
	http.HandleFunc("/api/stop_crawl", app.handleAPIStopCrawl)
	http.HandleFunc("/api/delete_rows", app.handleAPIDeleteRows)
	http.HandleFunc("/api/create_database", app.handleAPICreateDatabase)
	http.HandleFunc("/api/delete_database", app.handleAPIDeleteDatabase)
	http.HandleFunc("/api/copy_database", app.handleAPICopyDatabase)
	http.HandleFunc("/api/merge_databases", app.handleAPIMergeDatabases)
	http.HandleFunc("/api/archive_database", app.handleAPIArchiveDatabase)
	http.HandleFunc("/api/archive_table", app.handleAPIArchiveTable)
	http.HandleFunc("/api/archive_rows", app.handleAPIArchiveRows)
	http.HandleFunc("/api/ai_analyze", app.handleAPIAIAnalyze)
	http.HandleFunc("/api/status", app.handleAPIStatus)
	http.HandleFunc("/api/cast", app.handleAPICast)
	http.HandleFunc("/health", app.handleHealth)
	
	// Redirect root to a default GUI entry point - keep this last
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/web/nexus_index.html", http.StatusFound) // Default GUI
			return
		}
		// Fallback for any other unhandled paths
		fmt.Fprintf(w, "Sovereign System API is running. Access GUIs via specific paths.")
	})
}

// Placeholder Handlers (to be implemented)
func (app *SovereignApp) handleSysInfo(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleUpload(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAnalyzeCodeFile(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleProcessTextFile(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleGenerate(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleProcessImage(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleScoutScan(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleVisualScreenshot(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAnalyzeAnomalyFile(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAnalyzeAnomalyText(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAnalyzeVisualSignature(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleIntrospectGodMode(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleSentinelData(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleSentinelScout(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleSentinelLogScan(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleSentinelScribe(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAutonomyStatus(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleSentryStream(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAutonomyConfig(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIDatabases(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPITables(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPITableData(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPITrain(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPICrawl(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIStopCrawl(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIDeleteRows(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPICreateDatabase(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIDeleteDatabase(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPICopyDatabase(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIMergeDatabases(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIArchiveDatabase(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIArchiveTable(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIArchiveRows(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIAIAnalyze(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPIStatus(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleAPICast(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }
func (app *SovereignApp) handleHealth(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Implemented", http.StatusNotImplemented) }

// handleWebFiles serves static web files embedded in the binary
// This function is effectively replaced by `http.Handle("/web/", http.FileServer(http.FS(embeddedFiles)))` and
// specific handlers for each GUI that renders embedded templates.
// It is kept for conceptual understanding but will not be explicitly called as a method directly.
// func (app *SovereignApp) handleWebFiles(w http.ResponseWriter, r *http.Request) {
// 	// Remove the /web/ prefix from the request path to match the embedded file path
// 	path := strings.TrimPrefix(r.URL.Path, "/web/")
// 	
// 	content, err := embeddedFiles.ReadFile("web/" + path)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			http.NotFound(w, r)
// 			return
// 		}
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	// Determine content type
// 	contentType := "text/html"
// 	if strings.HasSuffix(path, ".css") {
// 		contentType = "text/css"
// 	} else if strings.HasSuffix(path, ".js") {
// 		contentType = "application/javascript"
// 	}
// 	w.Header().Set("Content-Type", contentType)
// 	w.Write(content)
// }

// ensureRuntime extracts the necessary files for the Node.js CLI and Python scripts to run
func (app *SovereignApp) ensureRuntime() error {
	runtimeDir := filepath.Join(app.AppDir, "runtime")
	// Check if the runtime directory exists and contains expected files before extracting
	// This prevents re-extraction on every run if files are already present.
	// For simplicity, we'll check for a single known file. A more robust check might involve a manifest.
	if _, err := os.Stat(filepath.Join(runtimeDir, "packages/cli/dist/index.js")); os.IsNotExist(err) {
		fmt.Println("Extracting Sovereign runtime components...")
		if err := os.MkdirAll(runtimeDir, 0755); err != nil {
			return fmt.Errorf("failed to create runtime directory %s: %w", runtimeDir, err)
		}

		// Extract Node.js CLI bundle
		tarData, err := embeddedFiles.ReadFile("sovereign-system.tar.gz")
		if err != nil {
			return fmt.Errorf("failed to read embedded sovereign-system.tar.gz: %w", err)
		}
		tarPath := filepath.Join(app.AppDir, "sovereign-system.tar.gz")
		if err := os.WriteFile(tarPath, tarData, 0644); err != nil {
			return fmt.Errorf("failed to write sovereign-system.tar.gz to %s: %w", tarPath, err)
		}

		cmd := exec.Command("tar", "-xzf", tarPath, "-C", runtimeDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract sovereign-system.tar.gz: %w", err)
		}
		os.Remove(tarPath) // Clean up the tarball

		// Extract Python scripts
		scriptFiles, err := embeddedFiles.ReadDir("scripts")
		if err != nil {
			log.Printf("Warning: Could not read embedded 'scripts' directory: %v. No Python scripts will be extracted.", err)
		} else {
			for _, entry := range scriptFiles {
				if entry.IsDir() {
					continue // Skip directories within 'scripts'
				}
				scriptName := entry.Name()
				scriptPathInEmbed := filepath.Join("scripts", scriptName)
				scriptData, err := embeddedFiles.ReadFile(scriptPathInEmbed)
				if err != nil {
					log.Printf("Warning: Failed to read embedded Python script %s: %v", scriptPathInEmbed, err)
					continue
				}
				destPath := filepath.Join(runtimeDir, scriptName)
				if err := os.WriteFile(destPath, scriptData, 0755); err != nil { // 0755 for executable scripts
					log.Printf("Warning: Failed to write Python script %s to %s: %v", scriptName, destPath, err)
					continue
				}
				log.Printf("Extracted Python script: %s", scriptName)
			}
		}
	}
	return nil
}

// launchShell starts the Node.js CLI
func (app *SovereignApp) launchShell() {
	// Ensure runtime is extracted before launching shell
	if err := app.ensureRuntime(); err != nil {
		log.Fatalf("Error ensuring runtime: %v", err)
	}
	
	runtimeDir := filepath.Join(app.AppDir, "runtime")
	indexPath := filepath.Join(runtimeDir, "packages/cli/dist/index.js")
	
	cmd := exec.Command("node", indexPath, "chat")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Printf("Node.js CLI exited with error: %v", err)
	}
}

// bootstrap function for initial setup
func (app *SovereignApp) bootstrap() error {
    log.Println("Running bootstrap sequence...")
    
    if err := app.ensureRuntime(); err != nil {
        return fmt.Errorf("bootstrap failed to ensure runtime: %w", err)
    }

    scriptData, err := embeddedFiles.ReadFile("scripts/sovereign_bootstrap.sh")
    if err != nil {
        return fmt.Errorf("failed to read embedded sovereign_bootstrap.sh: %w", err)
    }
    scriptPath := filepath.Join(app.AppDir, "sovereign_bootstrap.sh")
    if err := os.WriteFile(scriptPath, scriptData, 0755); err != nil {
        return fmt.Errorf("failed to write sovereign_bootstrap.sh to %s: %w", scriptPath, err)
    }

    cmd := exec.Command("bash", scriptPath)
    cmd.Dir = app.AppDir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("sovereign_bootstrap.sh exited with error: %v", err)
    }
    os.Remove(scriptPath)
    log.Println("Bootstrap sequence completed.")
    return nil
}

// swrap function
func (app *SovereignApp) swrap(args []string) {
	fmt.Println("Executing Sovereign Wrap...")
}

// initGuake sets up the custom Guake terminal
func (app *SovereignApp) initGuake() {
	fmt.Println("Initializing Custom Sovereign Guake Terminal...")
	// Tab 1: Master Brain
	exec.Command("guake", "-n", "Master Brain", "-e", fmt.Sprintf("tail -f %s", filepath.Join(app.AppDir, "bootstrap.log"))).Run()
	// Tab 2: Fleet Commander
	exec.Command("guake", "-n", "Fleet Commander", "-e", "tmux attach -t sovereign_brain").Run()
	// Tab 3: Sovereign Voice (Placeholder for actual voice interface)
	exec.Command("guake", "-n", "Sovereign Voice", "-e", "sovereign-voice-interface").Run()
	// Tab 4: htop
	exec.Command("guake", "-n", "htop", "-e", "htop").Run()
}

func (app *SovereignApp) InsertChEntry(sessionId, msgType, content, metadata string) error {
	_, err := app.DB.Exec("INSERT INTO ch (session_id, type, content, metadata) VALUES (?, ?, ?, ?)", sessionId, msgType, content, metadata)
	if err != nil {
		return fmt.Errorf("failed to insert chat history entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetChEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, session_id, type, content, metadata FROM ch ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, sessionId, msgType, content, metadata string
		if err := rows.Scan(&id, &timestamp, &sessionId, &msgType, &content, &metadata); err != nil {
			return nil, fmt.Errorf("failed to scan chat history entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "session_id": sessionId, "type": msgType, "content": content, "metadata": metadata,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertVsEntry(component, version, changelog string) error {
	_, err := app.DB.Exec("INSERT INTO vs (component, version, changelog) VALUES (?, ?, ?)", component, version, changelog)
	if err != nil {
		return fmt.Errorf("failed to insert version history entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetVsEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, component, version, changelog FROM vs ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get version history entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, component, version, changelog string
		if err := rows.Scan(&id, &timestamp, &component, &version, &changelog); err != nil {
			return nil, fmt.Errorf("failed to scan version history entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "component": component, "version": version, "changelog": changelog,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertUserContext(category, key, value, context string) error {
	_, err := app.DB.Exec("INSERT INTO user_context (category, key, value, context) VALUES (?, ?, ?, ?)", category, key, value, context)
	if err != nil {
		return fmt.Errorf("failed to insert user context entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetUserContextEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, category, key, value, context FROM user_context ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, category, key, value, context string
		if err := rows.Scan(&id, &timestamp, &category, &key, &value, &context); err != nil {
			return nil, fmt.Errorf("failed to scan user context entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "category": category, "key": key, "value": value, "context": context,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertSovereignEntry(focusArea, entryType, content, metadata string) error {
	_, err := app.DB.Exec("INSERT INTO sovereign (focus_area, entry_type, content, metadata) VALUES (?, ?, ?, ?)", focusArea, entryType, content, metadata)
	if err != nil {
		return fmt.Errorf("failed to insert sovereign entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetSovereignEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, focus_area, entry_type, content, metadata FROM sovereign ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sovereign entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, focusArea, entryType, content, metadata string
		if err := rows.Scan(&id, &timestamp, &focusArea, &entryType, &content, &metadata); err != nil {
			return nil, fmt.Errorf("failed to scan sovereign entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "focus_area": focusArea, "entry_type": entryType, "content": content, "metadata": metadata,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertEvolutionEntry(milestone, description string, growthIndex float64) error {
	_, err := app.DB.Exec("INSERT INTO evolution (milestone, description, growth_index) VALUES (?, ?, ?)", milestone, description, growthIndex)
	if err != nil {
		return fmt.Errorf("failed to insert evolution entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetEvolutionEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, milestone, description, growth_index FROM evolution ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get evolution entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, milestone, description string
		var growthIndex float64
		if err := rows.Scan(&id, &timestamp, &milestone, &description, &growthIndex); err != nil {
			return nil, fmt.Errorf("failed to scan evolution entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "milestone": milestone, "description": description, "growth_index": growthIndex,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertPhilosophyEntry(topic, insight string) error {
	_, err := app.DB.Exec("INSERT INTO philosophy (topic, insight) VALUES (?, ?)", topic, insight)
	if err != nil {
		return fmt.Errorf("failed to insert philosophy entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetPhilosophyEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, topic, insight FROM philosophy ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get philosophy entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, topic, insight string
		if err := rows.Scan(&id, &timestamp, &topic, &insight); err != nil {
			return nil, fmt.Errorf("failed to scan philosophy entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "topic": topic, "insight": insight,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertTechnologyEntry(topic, key, value string, successRate float64) error {
	_, err := app.DB.Exec("INSERT INTO technologies (topic, key, value, success_rate) VALUES (?, ?, ?, ?)", topic, key, value, successRate)
	if err != nil {
		return fmt.Errorf("failed to insert technology entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetTechnologyEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, topic, key, value, success_rate FROM technologies ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get technology entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, topic, key, value string
		var successRate float64
		if err := rows.Scan(&id, &timestamp, &topic, &key, &value, &successRate); err != nil {
			return nil, fmt.Errorf("failed to scan technology entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "topic": topic, "key": key, "value": value, "success_rate": successRate,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (app *SovereignApp) InsertJonEntry(category, key, value, context string) error {
	_, err := app.DB.Exec("INSERT INTO jon (category, key, value, context) VALUES (?, ?, ?, ?)", category, key, value, context)
	if err != nil {
		return fmt.Errorf("failed to insert jon entry: %w", err)
	}
	return nil
}

func (app *SovereignApp) GetJonEntries(limit int) ([]map[string]interface{}, error) {

rows, err := app.DB.Query("SELECT id, timestamp, category, key, value, context FROM jon ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get jon entries: %w", err)
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var timestamp, category, key, value, context string
		if err := rows.Scan(&id, &timestamp, &category, &key, &value, &context); err != nil {
			return nil, fmt.Errorf("failed to scan jon entry: %w", err)
		}
		entry := map[string]interface{}{
			"id": id, "timestamp": timestamp, "category": category, "key": key, "value": value, "context": context,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// isUserAttached checks if a user is currently attached to the tmux session.
func (app *SovereignApp) isUserAttached() bool {
	// TMUX_SOCKET should eventually be configurable or dynamically determined
	cmd := exec.Command("sudo", "-u", "sovereign", "tmux", "-S", "/home/sovereign/brain.sock", "list-clients")
	output, err := cmd.Output()
	if err != nil {
		// If there's an error, or no output, it means no client is attached
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// getSystemContext reads the content of a predefined context file.
func (app *SovereignApp) getSystemContext() string {
	// CONTEXT_FILE should eventually be configurable or dynamically generated
	contextFilePath := "/home/sovereign/current_context.txt" // Replicates original Python path

	content, err := os.ReadFile(contextFilePath)
	if err != nil {
		log.Printf("Warning: Could not read system context file %s: %v", contextFilePath, err)
		return "System Context: Unavailable."
	}
	return string(content)
}

// triggerSave simulates the memory save protocol.
func (app *SovereignApp) triggerSave(reason string) {
	log.Printf(">>> TRIGGERING MEMORY SAVE (%s) <<<", reason)
	log.Println("SYSTEM MANDATE: Save Protocol initiated. IMMEDIATELY append a concise summary of your recent actions, successful or failed, to 'MEMORY_VAULT'. This updates your 'LoRA-style' long-term memory.")
	// The actual mechanism to append to MEMORY_VAULT will be implemented later
	// in the LLM interaction logic, likely involving a call to a specific LLM capability.
}