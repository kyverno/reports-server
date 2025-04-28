package db

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresConfigStringPasswordWithSpecialCharacters(t *testing.T) {
	config := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "Agdg151[tr@y^q]9",
		DBname:   "testdb",
		SSLMode:  "disable",
	}

	connString := config.String()

	assert.Contains(t, connString, "postgres://", "Scheme missing")
	assert.Contains(t, connString, "@localhost:5432/testdb", "Host/Port/DB Name incorrect")
	assert.Contains(t, connString, "sslmode=disable", "SSLMode incorrect")

	parsedURL, err := url.Parse(connString)
	require.NoError(t, err, "Generated connection string should be parseable")

	require.NotNil(t, parsedURL.User, "Parsed URL should have Userinfo")
	assert.Equal(t, "testuser", parsedURL.User.Username(), "Username should match")

	pw, isSet := parsedURL.User.Password()
	assert.True(t, isSet, "Password should be set in parsed URL")
	assert.Equal(t, "Agdg151[tr@y^q]9", pw, "Decoded password should match original")

	expectedEncodedPassword := "Agdg151%5Btr%40y%5Eq%5D9" // #nosec G101
	expectedUserInfo := "testuser:" + expectedEncodedPassword
	assert.Equal(t, expectedUserInfo, parsedURL.User.String(), "Encoded user info string mismatch")

	assert.Equal(t, "localhost:5432", parsedURL.Host)
	assert.Equal(t, "/testdb", parsedURL.Path)
	assert.Equal(t, "disable", parsedURL.Query().Get("sslmode"))
}

func TestPostgresConfigStringSimplePassword(t *testing.T) {
	config := PostgresConfig{
		Host:     "192.168.1.100",
		Port:     6432,
		User:     "simpleuser",
		Password: "simplepassword",
		DBname:   "mydb",
		SSLMode:  "require",
	}

	connString := config.String()

	assert.Equal(t, "postgres://simpleuser:simplepassword@192.168.1.100:6432/mydb?sslmode=require", connString)

	_, err := url.Parse(connString)
	require.NoError(t, err, "Generated connection string should be parseable")
}

func TestPostgresConfigStringMultipleHosts(t *testing.T) {
	config := PostgresConfig{
		Host:     "host1,host2,host3",
		Port:     5432,
		User:     "multiuser",
		Password: "multipass",
		DBname:   "multidb",
		SSLMode:  "disable",
	}

	connString := config.String()

	expectedHostPart := "host1:5432,host2:5432,host3:5432"
	expectedURL := "postgres://multiuser:multipass@" + expectedHostPart + "/multidb?sslmode=disable"

	assert.Equal(t, expectedURL, connString)

	_, err := url.Parse(connString)
	require.NoError(t, err, "Generated connection string should be parseable")

	assert.Contains(t, connString, "@"+expectedHostPart+"/", "Multi-host string incorrect")
}

func TestPostgresConnection_SimplePassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}

	ctx := context.Background()
	simplePassword := "simplepass123"
	dbName := "users_simple"
	user := "testuser_simple"

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(user),
		postgres.WithPassword(simplePassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err, "Failed to start postgres container for simple password test")

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")
	portNat, err := pgContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err, "Failed to get container mapped port")
	port := portNat.Int()

	testConfig := PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: simplePassword,
		DBname:   dbName,
		SSLMode:  "disable",
	}

	connStr := testConfig.String()
	t.Logf("Generated connection string (simple password): %s", connStr)

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "sql.Open failed with generated connection string (simple password)")
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = db.PingContext(pingCtx)
	require.NoError(t, err, "db.PingContext failed using generated connection string (simple password)")
	t.Log("Successfully connected to PostgreSQL container using PostgresConfig.String() with simple password")
	db.Close()

	// --- Phase 2: Test connection with WRONG password ---
	t.Log("Attempting connection with wrong password (simple test)")
	wrongPassword := "wrongsimplepassXYZ"
	configWrongPw := PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: wrongPassword,
		DBname:   dbName,
		SSLMode:  "disable",
	}
	connStrWrongPw := configWrongPw.String()
	dbWrongPw, err := sql.Open("pgx", connStrWrongPw)
	require.NoError(t, err, "sql.Open should not fail immediately on wrong password (simple test)")
	defer dbWrongPw.Close()

	pingCtxWrong, cancelWrong := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWrong()
	err = dbWrongPw.PingContext(pingCtxWrong)
	require.Error(t, err, "db.PingContext SHOULD FAIL with wrong password (simple test)")
	t.Logf("Received expected error for wrong simple password: %v", err)
}

func TestPostgresConnection_SpecialCharsPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}

	ctx := context.Background()
	passwordWithSpecialChars := "Agdg151[tr@y^q]9" // #nosec G101
	dbName := "users_special"
	user := "testuser_special"

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(user),
		postgres.WithPassword(passwordWithSpecialChars),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
	)
	require.NoError(t, err, "Failed to start postgres container for special chars password test")

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")
	portNat, err := pgContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err, "Failed to get container mapped port")
	port := portNat.Int()

	testConfig := PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: passwordWithSpecialChars,
		DBname:   dbName,
		SSLMode:  "disable",
	}

	connStr := testConfig.String()
	t.Logf("Generated connection string (special chars password): %s", connStr)

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "sql.Open failed with generated connection string (special chars password)")
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = db.PingContext(pingCtx)
	require.NoError(t, err, "db.PingContext failed using generated connection string (special chars password)")
	t.Log("Successfully connected to PostgreSQL container using PostgresConfig.String() with special chars password")
	db.Close()

	// --- Phase 2: Test connection with WRONG password ---
	t.Log("Attempting connection with wrong password (special chars test)")
	wrongPassword := "wrongspecialpass!@#$%^"
	configWrongPw := PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: wrongPassword,
		DBname:   dbName,
		SSLMode:  "disable",
	}
	connStrWrongPw := configWrongPw.String()
	dbWrongPw, err := sql.Open("pgx", connStrWrongPw)
	require.NoError(t, err, "sql.Open should not fail immediately on wrong password (special chars test)")
	defer dbWrongPw.Close()

	pingCtxWrong, cancelWrong := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWrong()
	err = dbWrongPw.PingContext(pingCtxWrong)
	require.Error(t, err, "db.PingContext SHOULD FAIL with wrong password (special chars test)")
	t.Logf("Received expected error for wrong special chars password: %v", err)
}
