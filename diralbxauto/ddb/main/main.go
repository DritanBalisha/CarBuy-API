package main

import (
	"drialbXauto/pkg/config"
	"drialbXauto/pkg/routes"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Obtain the database connection using the GetDB function from your config package
	db := config.GetDB()

	// Create a new router
	r := mux.NewRouter()

	// Register your routes
	routes.RegisterCarRoutes(r)
	routes.AuthRoutes(r, db)
	routes.UserRoutes(r)
	routes.FilterRoutes(r)

	// Use the router as the handler for your HTTP server
	http.Handle("/", r)

	// Start the HTTP server
	log.Fatal(http.ListenAndServe("localhost:1010", r))
	// http.ListenAndServe("localhost:1010", context.ClearHandler(http.DefaultServeMux))
}
