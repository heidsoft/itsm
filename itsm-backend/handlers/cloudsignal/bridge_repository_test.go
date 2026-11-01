package cloudsignal

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestFindIncidentByCloudAlertID(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewBridgeRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("metadata->>'cloudAlertId' = $2")).
		WithArgs(int(7), "log-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(88)))

	id, err := repo.FindIncidentByCloudAlertID(context.Background(), 7, "log-1")
	require.NoError(t, err)
	require.Equal(t, int64(88), id)
}

func TestFindIncidentByCloudAlertIDMissingReturnsZero(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewBridgeRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("metadata->>'cloudAlertId' = $2")).
		WithArgs(int(7), "log-404").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	id, err := repo.FindIncidentByCloudAlertID(context.Background(), 7, "log-404")
	require.NoError(t, err)
	require.Zero(t, id)
}

func TestFindCIByCloudResourceUsesProviderPredicate(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewBridgeRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("cloud_provider = $3")).
		WithArgs(int(7), "i-bp1", "aliyun").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	id, err := repo.FindCIByCloudResource(context.Background(), 7, "aliyun", "i-bp1")
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
}

func TestFindCIByCloudResourceMissingReturnsZero(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewBridgeRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("cloud_resource_id = $2")).
		WithArgs(int(7), "i-gone", "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	id, err := repo.FindCIByCloudResource(context.Background(), 7, "", "i-gone")
	require.NoError(t, err)
	require.Zero(t, id)
}

func TestBridgeRepositoryValidatesTenantScope(t *testing.T) {
	db, _, done := newMock(t)
	defer done()

	repo := NewBridgeRepository(db)
	_, err := repo.FindIncidentByCloudAlertID(context.Background(), 0, "log-1")
	require.Error(t, err)
	_, err = repo.FindCIByCloudResource(context.Background(), 7, "aliyun", " ")
	require.Error(t, err)
}
