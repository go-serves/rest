# rest
<p align="center">
<img src="assets/images/resting-gopher.png" width="200" height="300" alt="Resting Gopher">
</p><br>

`rest` is a Go package that provides a production-ready HTTP server with built-in observability, configuration management, and graceful shutdown capabilities.

## Features

- **Graceful Shutdown**: Handles OS signals (SIGINT, SIGTERM) to shut down the server gracefully, ensuring all active requests are completed (up to a timeout).
- **Observability**:
    - **Prometheus Metrics**: Exposes a dedicated `/metrics` endpoint on a separate port/goroutine.
    - **RED Method**: Includes middleware to automatically instrument requests with Rate, Errors, and Duration metrics.
- **Configuration**: Easy configuration via environment variables using  [`envconfig`](https://github.com/kelseyhightower/envconfig).
- **Health Check**: Built-in `/health` endpoint.
- **Structured Logging**: Uses `log/slog` for structured logging.

## Usage

Here's a basic example of how to use `rest`:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/rabellamy/server/rest"
)

func main() {
	// 1. Create Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 2. Load Configuration (prefix 'test' means env vars like TEST_APIHOST)
	config, err := rest.LoadConfig("test")
	if err != nil {
		logger.Error("config loading failed", "err", err)
		os.Exit(1)
	}

	// 3. Define Routes
	routes := rest.Routes{
		"/hello": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, World!")
		},
	}

	// 4. Create Server
	server, err := rest.NewServer(context.Background(), config, routes, logger)
	if err != nil {
		logger.Error("server instantiation failed", "err", err)
		os.Exit(1)
	}

	// 5. Run Server
	if err := server.Run(); err != nil {
		logger.Error("server startup failed", "err", err)
		os.Exit(1)
	}
}
```

## Configuration

The server is configured using environment variables.

| Field | Environment Variable | Default | Description |
|-------|--------------------------------------|---------|-------------|
| `ReadTimeout` | `APP_READTIMEOUT` | `5s` | Maximum duration for reading the entire request. |
| `WriteTimeout` | `APP_WRITETIMEOUT` | `10s` | Maximum duration before timing out writes of the response. |
| `IdleTimeout` | `APP_IDLETIMEOUT` | `120s` | Maximum amount of time to wait for the next request when keep-alives are enabled. |
| `ShutdownTimeout` | `APP_SHUTDOWNTIMEOUT` | `20s` | Maximum duration to wait for graceful shutdown. |
| `APIHost` | `APP_APIHOST` | `0.0.0.0:3000` | Host and port for the main API server. |
| `DebugHost` | `APP_DEBUGHOST` | `0.0.0.0:3010` | Host and port for debug endpoints (if used). |
| `MetricsHost` | `APP_METRICSHOST` | `0.0.0.0:2112` | Host and port for the Prometheus metrics server. |
| `CorsAllowedOrigins` | `APP_CORSALLOWEDORIGINS` | `*` | List of allowed CORS origins. |
| `MaxHeaderBytes` | `APP_MAXHEADERBYTES` | `0` | Maximum number of bytes the server will read parsing the request header's keys and values. |
| `Build` | `APP_BUILD` | `dev` | Build version/tag. |
| `Desc` | `APP_DESC` | `example server` | Server description. |
| `Namespace` | `APP_NAMESPACE` | `APP` | Namespace for metrics. |

## Metrics

The server exposes Prometheus metrics at `http://<MetricsHost>/metrics` (default: `http://0.0.0.0:2112/metrics`).

Standard RED metrics (Rate, Errors, Duration) for your registered routes.
