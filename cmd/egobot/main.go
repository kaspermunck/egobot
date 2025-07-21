package main

import (
	"context"
	"net/http"
	"os"

	"egobot/internal/ai"
	"egobot/internal/pdf"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	r.POST("/extract", func(c *gin.Context) {
		var req struct {
			Entities []string `json:"entities"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}
		f, err := os.Open("statstidende_sample.pdf")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()
		text, err := pdf.ExtractText(f)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result, err := ai.ExtractEntities(context.Background(), text, req.Entities)
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
