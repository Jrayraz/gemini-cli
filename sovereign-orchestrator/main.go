package main

import (
	"embed"
	"flag" // Import the flag package
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // Still needed for database/sql
)

//go:embed core_directives.txt
//go:embed scripts/*
//go:embed sovereign-system.tar.gz
//go:embed web/* // Embed web assets
var embeddedFiles embed.FS

const (
	appName    = "sovereign"
	dbFileName = "sovereign_memory.db"
)

func main() {
	var customDBPath string
	flag.StringVar(&customDBPath, "db-path", "", "Path to an existing sovereign_memory.db file")
	flag.Parse()

	app, err := NewSovereignApp(customDBPath) // Pass customDBPath to NewSovereignApp
	if err != nil {
		log.Fatalf("Failed to create SovereignApp: %v", err)
	}
	defer app.Close() // Ensure DB connection is closed

	// Handle command-line arguments for specific actions
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init-guake":
			app.initGuake()
			return
		case "bootstrap":
			if err := app.bootstrap(); err != nil {
				log.Fatalf("Bootstrap failed: %v", err)
			}
			return
		case "swrap":
			app.swrap(os.Args[2:])
			return
		}
	}

	// Default application startup
	if err := app.Init(); err != nil {
		log.Fatalf("Failed to initialize SovereignApp: %v", err)
	}

	app.Run()
}
