# Store tests

## Database Support

This project supports PostgreSQL and SQLite for testing. MySQL support has been removed.

## How to test store with PostgreSQL?

1. Create a database in your PostgreSQL server.
2. Run the following command with two environment variables set:

```bash
DRIVER=postgres DSN="postgres://user:password@localhost:5432/memos_test?sslmode=disable" go test -v ./store/test/...
```

## How to test store with SQLite?

```bash
DRIVER=sqlite go test -v ./store/test/...
```

SQLite tests use temporary databases and require no external setup.
