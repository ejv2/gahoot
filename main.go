package main

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ethanv2/gahoot/config"
	"github.com/ethanv2/gahoot/game"
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
	Config      config.Config
	Coordinator game.GameCoordinator
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

	// Seed random
	// MUST be done before game coordinator
	rand.Seed(time.Now().UnixMilli())

	// Init configs
	Config, err = config.New(PathConfig)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatal("config not found")
		}
		log.Fatal(err)
	}

	// Init game coordinator
	Coordinator = game.NewCoordinator(Config.GameTimeout)
	log.Println("Generated test game", Coordinator.CreateGame().PIN)

	// Banner
	log.Printf("Gahoot! v%d.%d.%d server starting...", MajorVersion, MinorVersion, PatchVersion)
	log.Printf("Server listening on %s", Config.FullAddr())

	// Startup and listen
	router := gin.New()
	srv := http.Server{
		Addr:              Config.FullAddr(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
	router.Use(gin.Logger(), gin.Recovery())
	router.SetTrustedProxies(nil)

	router.LoadHTMLGlob("frontend/templates/*")
	router.Static("/static/", PathStatic)

	// Debugging specific router settings
	if gin.IsDebugging() {
		// Serve source files for debugging
		router.Static("/src/", "frontend/src/")
	}

	router.GET("/", handleRoot)
	router.GET("/join", handleJoin)

	create := router.Group("/create/")
	{
		create.GET("/", handleCreate)
		create.GET("/find", handleFind)
		create.GET("/upload", handleUpload)
		create.GET("/new", handleEditor)
	}

	play := router.Group("/play/")
	{
		play.GET("/game/:pin", handleGame)
		play.GET("/host/:pin", handleHost)
	}

	api := router.Group("/api/")
	{
		api.GET("/play/:pin", handlePlayApi)
		api.GET("/host/:pin", handleHostApi)
	}

	err = srv.ListenAndServe()
	log.Panic(err) // NOTREACHED: unless fatal error
}
