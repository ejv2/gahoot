package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/ejv2/gahoot/config"
	"github.com/ejv2/gahoot/game"
	"github.com/ejv2/gahoot/game/quiz"
)

// Core application paths.
const (
	PathFrontend  = "frontend"
	PathTemplates = PathFrontend + string(os.PathSeparator) + "templates"
	PathStatic    = PathFrontend + string(os.PathSeparator) + "static"
	PathConfig    = "config.gahoot"
)

// Application lifetime state.
var (
	Config      config.Config
	Coordinator game.Coordinator
	QuizManager quiz.Manager

	vd *validator.Validate
)

// checkFrontend checks if the frontend directory is a valid, readable
// directory.
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
	vd = validator.New()
	Config, err = config.New(PathConfig, vd)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatal("config not found")
		}

		log.Fatalf("bad configuration:\n%s", config.FormatErrors(err))
	}

	// Init quizzes
	QuizManager = quiz.NewManager()
	qs, err := QuizManager.LoadDir(Config.QuizPath)
	if err != nil {
		if warns, ok := err.(quiz.LoadDirError); ok {
			for _, elem := range warns {
				log.Println("WARNING:", elem)
			}
		} else {
			log.Fatal("error loading quiz store:", err)
		}
	}

	// Init game coordinator
	Coordinator = game.NewCoordinator(Config.GameTimeout)

	// Banner
	log.Printf("Gahoot! v%d.%d.%d server starting...", MajorVersion, MinorVersion, PatchVersion)
	log.Printf("Server listening on %s", Config.FullAddr())
	if len(qs) > 0 {
		log.Printf("Loaded %d quizzes from disk", len(qs))
	}

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

	// Load user-specified proxy config
	if err := router.SetTrustedProxies(Config.TrustedProxies); err != nil {
		log.Fatal("invalid proxy entries:", err)
	}

	router.LoadHTMLGlob(PathTemplates + "/*")
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

		create.GET("/game/", handleBlankCreateGame)
		create.GET("/game/:hash", handleCreateGame)
	}

	play := router.Group("/play/")
	{
		play.GET("/game/:pin", handleGame)
		play.GET("/host/:pin", handleHost)
	}

	api := router.Group("/api/")
	{
		api.GET("/play/:pin", handlePlayAPI)
		api.GET("/host/:pin", handleHostAPI)
	}

	errchan := make(chan error, 1)
	sigchan := make(chan os.Signal, 1)

	signal.Notify(sigchan, os.Interrupt)
	go func() {
		err := srv.ListenAndServe()
		errchan <- err
	}()

	select {
	case <-sigchan:
		log.Println("Caught interrupt signal. Terminating gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			if err == ctx.Err() {
				log.Println("Shutdown timeout reached. Terminating forcefully...")
				return
			}
			log.Fatal(err)
		}
	case err := <-errchan:
		if err != http.ErrServerClosed {
			log.Panic(err) // NOTREACHED: unless fatal error
		}
	}
}
