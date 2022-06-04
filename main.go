package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ethanv2/gahoot/config"
)

// Core application paths
const (
	PathFrontend  = "frontend"
	PathTemplates = PathFrontend + string(os.PathSeparator) + "templates"
	PathStatic    = PathFrontend + string(os.PathSeparator) + "static"
	PathConfig    = "config.gahoot"
)

// Application lifetime state
var (
	Config config.Config
)

// checkFrontend checks if the frontend directory
// is a valid, readable directory
func checkFrontend() error {
	_, err := os.ReadDir(PathFrontend)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var err error

	// Check frontend
	if err := checkFrontend(); err != nil {
		log.Fatal(err)
	}

	// Init configs
	Config, err = config.New(PathConfig)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatal("config not found")
		}
		log.Fatal(err)
	}

	// Banner
	log.Printf("Gahoot! v%d.%d.%d server starting...", MajorVersion, MinorVersion, PatchVersion)
	log.Printf("Server listening on %s", Config.FullAddr())

	// Startup and listen
	router := gin.New()
	srv := http.Server{
		Addr:         Config.FullAddr(),
		Handler:      router,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 3 * time.Second,
	}
	router.Use(gin.Logger(), gin.Recovery())

	router.LoadHTMLGlob("frontend/templates/*")
	router.Static("/static/", PathStatic)

	// Source files for JS debugging
	if gin.IsDebugging() {
		router.Static("/src/", "frontend/src/")
	}

	router.GET("/", handleRoot)
	router.GET("/join", handleJoin)

	err = srv.ListenAndServe()
	log.Panic(err) // NOTREACHED: unless fatal error
}