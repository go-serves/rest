# rest
<p align="center">
<img src="assets/images/resting-gopher.png" width="200" height="300" alt="Resting Gopher">
</p><br>

`rest` is a Go package that provides a production-ready HTTP server with built-in observability, configuration management, and graceful shutdown capabilities. It enforces a **Schema-First Design**, leveraging Protobuf and gRPC-Gateway to guarantee that your API documentation always exactly matches your implementation.

## Features

- **Schema-First Architecture**: Strictly enforces 1:1 service boundaries. All API routes must be defined via Protobuf schemas, eliminating drift between API contracts and server implementations.
- **Graceful Shutdown**: Handles OS signals (SIGINT, SIGTERM) to shut down the server gracefully, ensuring all active requests are completed (up to a timeout).
- **Observability (OpenTelemetry)**:
    - **Metrics**: Automatically captures semantic HTTP metrics (request duration, payload sizes, active requests) via `otelhttp`, inherently supporting the RED (Rate, Errors, Duration) method.
    - **Tracing**: Configurable distributed tracing for all incoming HTTP requests via OTLP.
    - **Customizable Providers**: Supports injecting your own custom `MeterProvider` or `TracerProvider` via functional options (`WithMeterProvider`, `WithTracerProvider`).
- **Configuration**: Easy configuration via environment variables using [`envconfig`](https://github.com/kelseyhightower/envconfig).
- **Health Check**: Built-in `/health` endpoint.
- **Structured Logging**: Uses `log/slog` for structured logging.

## Usage

See [example](example/main.go)

## Configuration

The server is configured using environment variables. If you pass a prefix like `APP` to `LoadConfig`, prefix all environment variables below with `APP_` (e.g. `APP_APIHOST`).

| Field | Environment Variable | Default | Description |
|-------|--------------------------------------|---------|-------------|
| `ReadTimeout` | `READTIMEOUT` | `5s` | Maximum duration for reading the entire request. |
| `WriteTimeout` | `WRITETIMEOUT` | `10s` | Maximum duration before timing out writes of the response. |
| `IdleTimeout` | `IDLETIMEOUT` | `120s` | Maximum amount of time to wait for the next request when keep-alives are enabled. |
| `ShutdownTimeout` | `SHUTDOWNTIMEOUT` | `20s` | Maximum duration to wait for graceful shutdown. |
| `APIHost` | `APIHOST` | `0.0.0.0:3000` | Host and port for the main API server. |
| `DebugHost` | `DEBUGHOST` | `0.0.0.0:3010` | Host and port for debug endpoints (if used). |
| `MetricsHost` | `METRICSHOST` | `0.0.0.0:2112` | Host and port for the Prometheus metrics server. |
| `CorsAllowedOrigins` | `CORSALLOWEDORIGINS` | `*` | List of allowed CORS origins. |
| `MaxHeaderBytes` | `MAXHEADERBYTES` | `0` | Maximum number of bytes the server will read parsing the request header's keys and values. |
| `Build` | `BUILD` | `dev` | Build version/tag. |
| `Desc` | `DESC` | `example server` | Server description. |
| `Namespace` | `NAMESPACE` | `APP` | Namespace for metrics. |
| `TraceExporterEndpoint`| `TRACEEXPORTERENDPOINT` | `""` | The OTLP gRPC endpoint for trace exports (e.g. `localhost:4317`). Tracing is disabled if empty. |

## Metrics

The server exposes Prometheus metrics at `http://<MetricsHost>/metrics` (default: `http://0.0.0.0:2112/metrics`).
