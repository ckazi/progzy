package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"proxy-server/database"
	"proxy-server/handlers"
	"proxy-server/middleware"
	"proxy-server/proxy"
)

func main() {
	log.Println("Starting Proxy Server Application...")

	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "8080"
	}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8081"
	}

	proxyServer := proxy.NewProxyServer(db, proxyPort)
	go func() {
		log.Printf("Starting proxy server on port %s", proxyPort)
		if err := proxyServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy server failed: %v", err)
		}
	}()

	authHandler := handlers.NewAuthHandler(db)
	usersHandler := handlers.NewUsersHandler(db)
	statsHandler := handlers.NewStatsHandler(db)
	settingsHandler := handlers.NewSettingsHandler(db)
	scheduleLogCleanup(db)

	r := mux.NewRouter()

	r.HandleFunc("/api/init/check", authHandler.CheckInit).Methods("GET")
	r.HandleFunc("/api/init/setup", authHandler.InitSetup).Methods("POST")
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/api/auth/2fa/verify", authHandler.VerifyTwoFA).Methods("POST")

	authMiddleware := middleware.NewAuthMiddleware(db)

	twofa := r.PathPrefix("/api/auth/2fa").Subrouter()
	twofa.Use(authMiddleware.Handler)
	twofa.HandleFunc("/setup", authHandler.SetupTwoFA).Methods("POST")
	twofa.HandleFunc("/verify-setup", authHandler.VerifySetupTwoFA).Methods("POST")
	twofa.HandleFunc("/disable", authHandler.DisableTwoFA).Methods("POST")
	twofa.HandleFunc("/backup-codes", authHandler.RegenerateBackupCodes).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()
	api.Use(authMiddleware.Handler)
	api.Use(middleware.AdminMiddleware)

	api.HandleFunc("/stats/dashboard", statsHandler.GetDashboardStats).Methods("GET")
	api.HandleFunc("/stats/traffic", statsHandler.GetTrafficStats).Methods("GET")
	api.HandleFunc("/logs/requests", statsHandler.GetRequestLogs).Methods("GET")
	api.HandleFunc("/logs/requests/export", statsHandler.ExportRequestLogs).Methods("GET")
	api.HandleFunc("/logs/retention", statsHandler.GetLogRetention).Methods("GET")
	api.HandleFunc("/logs/clear", statsHandler.ClearRequestLogs).Methods("POST")
	api.HandleFunc("/audit/logs", statsHandler.GetAuditLogs).Methods("GET")

	api.HandleFunc("/users", usersHandler.GetAllUsers).Methods("GET")
	api.HandleFunc("/users", usersHandler.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", usersHandler.GetUser).Methods("GET")
	api.HandleFunc("/users/{id}", usersHandler.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", usersHandler.DeleteUser).Methods("DELETE")

	api.HandleFunc("/settings", settingsHandler.GetSettings).Methods("GET")
	api.HandleFunc("/settings", settingsHandler.UpdateSetting).Methods("PUT")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	log.Printf("Starting API server on port %s", apiPort)
	if err := http.ListenAndServe(":"+apiPort, handler); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func scheduleLogCleanup(db *database.Database) {
	go func() {
		runCleanup := func() {
			days, err := db.GetLogRetentionDays()
			if err != nil {
				log.Printf("Failed to get log retention settings: %v", err)
				return
			}
			if days <= 0 {
				return
			}
			if deleted, err := db.DeleteRequestLogsOlderThan(days); err != nil {
				log.Printf("Failed to cleanup logs: %v", err)
			} else if deleted > 0 {
				log.Printf("Cleaned up %d request log entries older than %d days", deleted, days)
			}
		}

		runCleanup()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			runCleanup()
		}
	}()
}
