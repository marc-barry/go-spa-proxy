package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const PORT int = 3000

var ADDR = strings.Join([]string{"", strconv.Itoa(PORT)}, ":")

type Config struct {
	Apps map[string]map[string]string `json:"apps"`
}

var config Config

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Array of headers to strip from the response
type stripHeaders [1]string

func (t stripHeaders) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	// Loop through each header that we want to strip and remove them from the response
	for _, c := range t {
		resp.Header.Del(c)
	}
	return resp, nil
}

func proxy(c *gin.Context) {
	remote, err := url.Parse(config.Apps[c.Request.Host]["uri"])
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	stripHeadersList := [...]string{"Cache-Control"}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = remote.Path + req.URL.Path
	}
	proxy.Transport = stripHeaders(stripHeadersList)

	proxy.ServeHTTP(c.Writer, c.Request)
}

func main() {
	configFile, err := os.Open("./config.json")
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	defer configFile.Close()
	configFileBytes, err := io.ReadAll(configFile)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	if err := json.Unmarshal(configFileBytes, &config); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "healthy")
	})

	router.GET("/readyz", func(c *gin.Context) {
		if _, err := os.Stat(getEnv("READY_FILE", "ready")); err == nil {
			c.String(200, "ready")
		} else {
			c.String(200, "not ready")
		}
	})

	router.GET("/_next/*filepath", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
		proxy(c)
	})

	router.GET("/static/*filepath", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
		proxy(c)
	})

	router.NoRoute(func(c *gin.Context) {
		c.Request.URL.Path = "/index.html"
		c.Writer.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		proxy(c)
	})

	router.Run(ADDR)
}
