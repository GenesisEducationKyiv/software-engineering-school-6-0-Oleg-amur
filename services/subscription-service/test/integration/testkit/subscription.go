//go:build integration

package testkit

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func AssertSubscriptionStatus(t *testing.T, db *sql.DB, token string, want int) {
	t.Helper()

	var got int
	err := db.QueryRowContext(
		t.Context(),
		`SELECT subscription_status FROM subscriptions WHERE token = $1`,
		token,
	).Scan(&got)
	require.NoError(t, err, "get subscription status by token")
	require.Equal(t, want, got)
}
