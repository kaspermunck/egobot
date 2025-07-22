package main

import (
	"context"
	"encoding/json"
	"net/http"

	"egobot/internal/ai"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	r.POST("/extract", func(c *gin.Context) {
		// Parse multipart form
		err := c.Request.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
			return
		}
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing PDF file: " + err.Error()})
			return
		}
		defer file.Close()

		entitiesStr := c.Request.FormValue("entities")
		if entitiesStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing entities field (should be a JSON array)"})
			return
		}
		var entities []string
		if err := json.Unmarshal([]byte(entitiesStr), &entities); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entities JSON: " + err.Error()})
			return
		}

		// Pass the file (as multipart.File) and filename to the AI extractor
		result, err := ai.ExtractEntitiesFromPDFFile(context.Background(), file, header.Filename, entities)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})
	return r
}

func RunServer(lc fx.Lifecycle, router *gin.Engine) {
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go server.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(NewRouter),
		fx.Invoke(RunServer),
	)
	app.Run()
}
