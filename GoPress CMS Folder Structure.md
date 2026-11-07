# gopress CMS – Initial Folder Structure

```
gopress/
├── cmd/
│   └── gopress-server/
│       ├── main.go          # Application entry point
│       └── config.go        # Configuration loading
│
├── internal/
│   ├── core/
│   │   ├── engine.go        # Core CMS engine
│   │   └── router.go        # Request routing
│   │
│   ├── handler/
│   │   ├── http/
│   │   │   ├── admin.go     # Admin panel handlers
│   │   │   ├── api.go       # REST API handlers
│   │   │   └── public.go    # Public site handlers
│   │   └── middleware/
│   │       ├── auth.go      # Authentication middleware
│   │       ├── logging.go   # Request logging
│   │       └── recovery.go  # Panic recovery
│   │
│   ├── service/
│   │   ├── content.go       # Content CRUD operations
│   │   ├── user.go          # User management
│   │   └── media.go         # Media/file handling
│   │
│   ├── model/
│   │   ├── content.go       # Content struct & DB schema
│   │   ├── user.go          # User struct & DB schema
│   │   └── media.go         # Media struct & DB schema
│   │
│   └── store/
│       ├── db.go            # Database connection setup
│       ├── content_store.go # Content persistence
│       ├── user_store.go    # User persistence
│       └── media_store.go   # Media persistence
│
├── pkg/
│   ├── database/
│   │   └── postgres.go      # PostgreSQL driver wrapper
│   ├── logger/
│   │   └── zap_logger.go    # Structured logging
│   └── util/
│       ├── hash.go          # Password hashing
│       └── uuid.go          # ID generation helpers
│
├── api/
│   └── v1/
│       ├── openapi.yaml     # OpenAPI/Swagger spec
│       └── routes.go        # API route definitions
│
├── web/
│   ├── static/
│   │   ├── css/             # Admin panel styles
│   │   ├── js/              # Frontend scripts
│   │   └── images/          # Static assets
│   └── templates/
│       ├── admin/           # Admin HTML templates
│       └── public/          # Public site templates
│
├── configs/
│   ├── development.yaml     # Dev environment config
│   ├── staging.yaml         # Staging config
│   └── production.yaml      # Production config
│
├── scripts/
│   ├── migrate.sh           # Database migrations runner
│   └── build.sh             # Build script
│
├── test/
│   ├── integration/         # Integration tests
│   │   ├── content_test.go
│   │   ├── user_test.go
│   │   └── api_test.go
│   └── fixtures/            # Test data/fixtures
│
├── .env.example             # Environment template
├── Makefile                 # Build/test commands
├── go.mod                   # Module definition
├── go.sum                   # Dependencies
└── README.md                # Project documentation
```

---

## Key Design Principles

**Simple and Idiomatic**
- Follows standard Go project layout conventions
- Easy to navigate and extend
- Clear separation of concerns

**Test-Ready**
- Dedicated `test/integration` folder for end-to-end tests
- Internal packages structured for unit test isolation
- Fixtures folder for reproducible test data

**Modular Growth**
- Each domain (content, user, media) has its own handler/service/store slice
- Middleware separated for easy composition
- Config environments isolated for safe deployments

---

## First Implementation Order

1. **Foundation** (`cmd/`, `go.mod`, `configs/`)
2. **Core Layer** (`internal/model/`, `internal/store/db.go`)
3. **Business Logic** (`internal/service/`)
4. **HTTP Interface** (`internal/handler/http/`, `internal/handler/middleware/`)
5. **Tests** (`test/integration/*_test.go`)
6. **Assets & Templates** (`web/`)
7. **Documentation** (`README.md`, `api/v1/openapi.yaml`)

---

## Example File Skeletons

### `cmd/gopress-server/main.go`
```go
package main

import (
    "log"
    "os"

    "github.com/xlthlx/gopress/internal/core"
    "github.com/xlthlx/gopress/pkg/logger"
)

func main() {
    log.Println("Starting gopress server...")

    // Load config
    cfg := loadConfig()

    // Setup logger
    logger.Init(cfg.LogLevel)

    // Initialize engine
    engine := core.NewEngine(cfg)

    // Run server
    if err := engine.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### `Makefile` (example targets)
```makefile
.PHONY: build test clean run

build:
    go build -o bin/gopress ./cmd/gopress-server

test:
    go test -v ./...

clean:
    rm -rf bin/

run:
    go run ./cmd/gopress-server
```

---

## Naming Convention Notes

- Package names are lowercase and descriptive (`store`, `handler`, `service`)
- Files within packages are named after their primary responsibility (`content.go`, `auth.go`)
- Test files mirror source: `content.go` → `content_test.go`
- Import path: `github.com/xlthlx/gopress/internal/...`

---

This structure scales well from a small CMS to a larger platform while staying true to your preference for simplicity. Once you start coding, the natural growth will be horizontal (adding new features in existing layers) rather than deep nesting.
