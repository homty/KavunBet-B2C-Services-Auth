# User service tests

Tests are grouped by the layer they exercise:

- `grpc`: in-memory gRPC transport, handler validation, and gRPC error mapping.
- `rest`: REST handler unit tests with a service spy.
- `service`: service unit tests with an in-memory repository.
- `repository`: PostgreSQL integration tests; these require `DATABASE_URL` and the current migrations.

From the repository root (`BetWin`), run all user-service tests with:

```sh
go test ./services/user-service/...
```

From the user-service module directory (`BetWin/services/user-service`), use:

```sh
go test ./...
```

Run only the fast, database-independent suites from the repository root:

```sh
go test ./services/user-service/tests/grpc ./services/user-service/tests/rest ./services/user-service/tests/service
```

Or, from the user-service module directory:

```sh
go test ./tests/grpc ./tests/rest ./tests/service
```
