package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/im7mortal/release-analyzer/pkg/releaseAnalyzer"
	"log"
	"net/http"
	"os"
)

type Response struct {
	Deltas []*releaseAnalyzer.Delta `json:"deltas"`
}

const defaultPort = "8080" // Default server port

func main() {
	r := gin.Default()

	r.SetTrustedProxies(nil)

	r.GET("/:org/:repo/bloat", bloatHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	log.Printf("Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// bloatHandler handles requests to analyze release bloat
func bloatHandler(c *gin.Context) {
	org := c.Param("org")
	repo := c.Param("repo")
	startTag := c.Query("start")
	endTag := c.Query("end")

	if org == "" || repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing organization or repository"})
		return
	}

	if startTag == "" || endTag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'start' or 'end' query parameters"})
		return
	}

	deltas, err := releaseAnalyzer.Process(c.Request.Context(), org, repo, startTag, endTag)

	if err != nil {
		var tagError *releaseAnalyzer.TagNotFound
		if errors.As(err, &tagError) {
			// Handle TagNotExist errors with 400 status code
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	response := Response{Deltas: deltas}
	c.JSON(http.StatusOK, response)
}
