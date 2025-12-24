package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/rabellamy/server/rest"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Hello from myHandler! You requested: %s", r.URL.Path)
}

func anotherHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Hello from anotherHandler! You requested: %s", r.URL.Path)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	config, err := rest.LoadConfig("test")
	if err != nil {
		logger.Error("server instantiation failed", "err", err)
		os.Exit(1)
	}

	routes := rest.Routes{
		"/myHandler":      myHandler,
		"/anotherHandler": anotherHandler,
	}

	server, err := rest.NewServer(context.Background(), config, routes, logger)
	if err != nil {
		logger.Error("server instantiation failed", "err", err)
		os.Exit(1)
	}

	if err := server.Run(); err != nil {
		logger.Error("server startup failed", "err", err)
		os.Exit(1)
	}
}
