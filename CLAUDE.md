# Minder Development Guide for Claude Code

## Project Overview

Minder is an open-source supply chain security platform that helps development teams build more secure software and prove their security posture. It enables proactive security policy management across repositories and artifacts, with features for continuous security enforcement, artifact attestation, and dependency management.

**Key Technologies:**
- **Language**: Go 1.24+
- **Protocol Buffers**: gRPC for API communication, REST via grpc-gateway
- **Database**: PostgreSQL with sqlc for type-safe SQL
- **Message Queue**: NATS JetStream for event-driven architecture
- **Authentication**: Keycloak (OAuth2/OIDC), JWT tokens
- **Authorization**: OpenFGA (relationship-based access control)
- **Frontend CLI**: Cobra framework with Bubble Tea TUI components
- **Observability**: OpenTelemetry, Prometheus metrics, zerolog
- **Security**: Sigstore for artifact signing/verification

## Architecture

Minder consists of:

1. **minder-server**: gRPC/REST API server (control plane)
2. **minder CLI**: Command-line interface for users
3. **reminder**: Background service for scheduled tasks
4. **Control Plane** (`internal/controlplane/`): API handlers and business logic
5. **Engine** (`internal/engine/`): Policy evaluation and enforcement
6. **Providers** (`internal/providers/`): Integration with GitHub, GitLab, container registries
7. **Datasources** (`internal/datasources/`): REST API data fetching and ingestion

## Directory Structure

```
.
├── cmd/                    # Main applications
│   ├── cli/               # minder CLI tool
│   ├── server/            # minder-server (gRPC/REST API)
│   ├── dev/               # Development utilities
│   └── reminder/          # Scheduled task service
├── internal/              # Private application code
│   ├── controlplane/      # API handlers and orchestration
│   ├── engine/            # Policy evaluation engine
│   ├── providers/         # GitHub, GitLab, container registry integrations
│   ├── datasources/       # REST API datasources
│   ├── db/                # Generated SQLC database code
│   ├── auth/              # Authentication (Keycloak, JWT)
│   ├── authz/             # Authorization (OpenFGA)
│   ├── events/            # Event handling (NATS)
│   ├── entities/          # Entity management
│   ├── reconcilers/       # State reconciliation
│   └── crypto/            # Cryptographic operations
├── pkg/                   # Public library code
│   ├── api/               # Generated protobuf/OpenAPI code
│   ├── profiles/          # Security profiles
│   ├── ruletypes/         # Rule type definitions
│   ├── mindpak/           # Package management
│   ├── engine/            # Engine interfaces
│   └── providers/         # Provider interfaces
├── proto/                 # Protocol buffer definitions
├── database/              # Database layer
│   ├── migrations/        # SQL migrations (golang-migrate)
│   ├── query/             # SQLC query definitions
│   └── schema/            # Database schema
├── deployment/            # Kubernetes and Helm charts
├── docs/                  # Documentation (Docusaurus)
├── examples/              # Example configurations
└── .mk/                   # Makefile includes
```

## Development Workflow

### Prerequisites

**Required tools:**
- Go 1.24+
- Docker & Docker Compose
- OpenSSL (for key generation)

**Build tools (installed via `make bootstrap`):**
- ko (for container builds)
- buf (Protocol Buffer compilation)
- sqlc (SQL code generation)
- golangci-lint (linting)
- gotestfmt (test output formatting)
- protoc plugins (grpc-gateway, protoc-gen-go, etc.)
- mockgen (mock generation)
- yq (YAML processing)
- fga (OpenFGA CLI)
- helm-docs (Helm documentation)

**Runtime services (via Docker):**
- PostgreSQL (database)
- Keycloak (authentication)
- NATS (message queue)

### Initial Setup

Before building or running Minder, install all build dependencies and initialize configuration:

```bash
# Install build tools and initialize configuration
make bootstrap
```

This command will:
- Install all Go-based build tools (sqlc, protoc plugins, mockgen, etc.)
- Create `config.yaml` and `server-config.yaml` from example templates (if they don't exist)
- Generate encryption keys in `.ssh/` directory for token signing

**Note**: Run `make bootstrap` once after cloning the repository. You may need to run it again if build tool versions change.

### Building

```bash
# Build both CLI and server binaries to ./bin/
make build

# Clean build artifacts
make clean
```

### Code Generation

Code generation is critical and must be run after changes to:

```bash
# Run all code generation (protobuf, sqlc, mocks, OpenAPI)
make gen

# Individual generators:
make buf        # Generate protobuf code from proto/
make sqlc       # Generate Go code from database/query/*.sql
make mock       # Generate mocks using mockgen
make oapi       # Generate OpenAPI client code
make cli-docs   # Generate CLI documentation
```

**When to regenerate:**
- After modifying `.proto` files → `make buf`
- After modifying `.sql` files in `database/query/` → `make sqlc`
- After adding database migrations → `make sqlc`
- After modifying interfaces that need mocks → `make mock`
- After changing CLI commands → `make cli-docs`

### Database Management

Minder uses PostgreSQL with golang-migrate for migrations and sqlc for type-safe queries.

```bash
# Start local database
make run-docker

# Run migrations
make migrateup

# Rollback migrations
make migratedown

# After adding queries to database/query/*.sql, regenerate Go code
make sqlc
```

**Database conventions:**
- Migrations in `database/migrations/` numbered sequentially
- Queries in `database/query/*.sql` with sqlc annotations
- Generated code in `internal/db/`
- Always use prepared statements via sqlc
- Use transactions for multi-statement operations

### Testing

```bash
# Run all tests with verbose output
make test

# Run tests silently (errors only)
make test-silent

# Run tests with coverage
make cover

# Coverage in silent mode
make test-cover-silent

# Lint Go code
make lint-go

# Lint protobuf definitions
make lint-buf

# Run all linters
make lint
```

**Testing conventions:**
- Test files: `*_test.go` alongside source files
- Use `testify/assert` for assertions
- Use `testify/require` for setup/prerequisites
- Use `testify/mock` or `go.uber.org/mock` for mocking
- Use `testify/suite` for integration tests
- Mock database queries via `database/mock/store.go`
- Parallel tests encouraged with `t.Parallel()`
- Coverage excludes: auto-generated code (db, proto, mocks)

### Running Locally

```bash
# Start all services (server, Postgres, Keycloak, NATS)
make run-docker

# Configure GitHub OAuth for Keycloak
make KC_GITHUB_CLIENT_ID=<id> KC_GITHUB_CLIENT_SECRET=<secret> github-login

# Use CLI against local server (requires config.yaml)
cp config/config.yaml.example config.yaml
minder auth login  # defaults to localhost:8090

# Use CLI against hosted instance
minder auth login --grpc-host api.custcodian.dev
```

**Configuration:**
- Server config: `server-config.yaml` (from `config/server-config.yaml.example`)
- CLI config: `config.yaml` (from `config/config.yaml.example`)
- Environment-specific configs in `config/` directory
- Use `MINDER_CONFIG` env var to specify config file

## Code Style and Conventions

### Go Code Standards

- **Logging**: Use `github.com/rs/zerolog` (NOT standard `log` package)
- **Error handling**: Wrap errors with context using `fmt.Errorf` with `%w`
- **Naming**: Follow Go conventions (PascalCase exports, camelCase private)
- **Package structure**: Follow standard Go project layout
- **Comments**: Document all exported functions, types, and packages
- **Linting**: All code must pass `make lint` before commit
- **Cyclomatic complexity**: Keep functions under complexity 15
- **Line length**: Max 120 characters (enforced by linter)

### Protobuf Conventions

- Definitions in `proto/minder/v1/`
- Use buf for linting and generation
- Follow buf style guide
- All RPCs must have `RpcOptions` with appropriate authorization
- Use `google.api.annotations` for REST mapping
- Validate inputs with `buf.validate.validate` constraints

### Database Conventions

- Use sqlc for all database access
- Write queries in `database/query/*.sql`
- Use named parameters: `@param_name`
- Prefer `RETURNING *` for INSERT/UPDATE operations
- Use transactions via `db.WithTransaction()`
- Never use raw SQL strings in code
- All timestamps should be `timestamptz`

### Git Commit Messages

Follow Chris Beams' guide:
1. Separate subject from body with blank line
2. Limit subject to 50 characters
3. Capitalize subject line
4. No period at end of subject
5. Use imperative mood ("Add feature" not "Added feature")
6. Explain what and why in body, not how

Example:
```
Add secret scanning remediation support

Implements automatic remediation for secret scanning alerts by
enabling the feature through the GitHub API when profiles are
evaluated. This ensures repositories stay compliant with security
policies without manual intervention.
```

## Key Patterns and Practices

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process entity %s: %w", entityID, err)
}

// Bad: Bare return
if err != nil {
    return err
}
```

### Logging

```go
// Good: Use zerolog with structured fields
zerolog.Ctx(ctx).Info().
    Str("entity_id", entityID).
    Str("rule_type", ruleType).
    Msg("evaluating rule")

// Bad: Use standard log
log.Printf("evaluating rule for %s", entityID)
```

### Database Access

```go
// Good: Use sqlc-generated queries
repo, err := s.store.GetRepositoryByID(ctx, repoID)
if err != nil {
    return fmt.Errorf("failed to get repository: %w", err)
}

// Good: Use transactions
err := s.store.WithTransaction(func(qtx db.ExtendedQuerier) error {
    // Multiple operations
    return nil
})
```

### Testing with Mocks

```go
// Use mockgen-generated mocks
mockStore := mockdb.NewMockStore(ctrl)
mockStore.EXPECT().
    GetRepositoryByID(gomock.Any(), repoID).
    Return(db.Repository{}, nil)
```

### gRPC Server Implementation

```go
// Always validate authorization
if err := s.authz.CheckAuthorization(ctx, req); err != nil {
    return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
}

// Use proper status codes
if errors.Is(err, sql.ErrNoRows) {
    return nil, status.Error(codes.NotFound, "resource not found")
}
```

## Common Development Tasks

### Adding a New gRPC Endpoint

1. Define RPC in `proto/minder/v1/minder.proto`
2. Add `RpcOptions` with authorization relation
3. Run `make buf` to generate Go code
4. Implement handler in `internal/controlplane/handlers_*.go`
5. Add authorization check in handler
6. Add tests in `internal/controlplane/*_test.go`
7. Update documentation

### Adding a Database Table

1. Create migration files manually in `database/migrations/` (numbered sequentially)
2. Write UP and DOWN SQL in `database/migrations/`
3. Add queries in `database/query/foo.sql`
4. Run `make sqlc` to generate Go code
5. Use generated functions in `internal/db/`
6. Write tests using mock store

### Adding a New Rule Type

1. Define rule type schema (YAML or JSON)
2. Implement evaluator in `internal/engine/eval/`
3. Add remediation handler in `internal/engine/actions/remediate/`
4. Add alert handler in `internal/engine/actions/alert/`
5. Register rule type in engine
6. Add integration tests
7. Document in `docs/`

### Adding Provider Support

1. Implement provider interface in `internal/providers/`
2. Add OAuth configuration in provider
3. Implement entity fetching (repos, artifacts, PRs)
4. Add provider registration in control plane
5. Add CLI commands in `cmd/cli/app/provider/`
6. Add integration tests
7. Document provider-specific features

## Debugging Tips

### Local Development

- Server logs: Check Docker Compose output
- Database: Use `psql` to inspect database directly
- gRPC: Use `grpcurl` for manual API testing
- Events: Check NATS JetStream logs for event flow
- Auth: Check Keycloak admin console at `localhost:8081`

### Common Issues

**"Permission denied" errors**: Check OpenFGA authorization model and user roles
**Database migration errors**: Ensure migrations are sequentially numbered and reversible
**gRPC connection refused**: Ensure server is running and config points to correct host:port
**Token expired**: Run `minder auth login` to refresh authentication
**Code generation out of sync**: Run `make gen` to regenerate all code

## Useful Make Targets

```bash
make help           # Show all available targets
make bootstrap      # Install build tools and initialize configuration (run once)
make build          # Build CLI and server binaries
make gen            # Run all code generators
make test           # Run all tests with verbose output
make cover          # Run tests with coverage report
make lint           # Run all linters
make clean          # Clean generated files and binaries
make run-docker     # Start all services locally
make migrateup      # Apply database migrations
make migratedown    # Rollback one migration
make cli-docs       # Generate CLI documentation
```

## Additional Resources

- **Documentation**: https://mindersec.github.io/
- **API Reference**: https://mindersec.github.io/ref/api
- **Proto Reference**: https://mindersec.github.io/ref/proto
- **CLI Reference**: https://mindersec.github.io/ref/cli/minder
- **Rules & Profiles**: https://github.com/mindersec/minder-rules-and-profiles
- **Discord**: https://discord.gg/RkzVuTp3WK
- **Contributing**: See CONTRIBUTING.md

## Security Considerations

- Never commit secrets, API keys, or tokens
- Use `.env` files for local secrets (already in `.gitignore`)
- All API endpoints require authentication and authorization
- Implement rate limiting for public-facing APIs
- Validate all user inputs with protobuf validators
- Use parameterized queries (sqlc) to prevent SQL injection
- Follow SLSA Build Level 3 practices for releases
- Sign all release artifacts with Sigstore

## Notes for AI Assistants

- Always run `make gen` after modifying `.proto` or `.sql` files
- Check authorization in all gRPC handlers before business logic
- Use zerolog for logging, never standard library `log`
- Write tests for new features before marking work complete
- Follow the commit message guidelines strictly
- Update documentation in `docs/` for user-facing changes
- Mock external dependencies in tests (GitHub API, etc.)
- Use `internal/db` for database access, never raw SQL
- Check `make lint` passes before considering code complete
- When adding dependencies, run `go mod tidy` and commit `go.mod`/`go.sum`
