//go:build integration

package dialog_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/internal/auth"
	"vn_startup_core_backend_go/internal/dialog"
	"vn_startup_core_backend_go/internal/testutil"
	"vn_startup_core_backend_go/internal/wallet"
)

type poolSceneProvider struct{ pool *pgxpool.Pool }

func (p poolSceneProvider) GetSceneInfo(ctx context.Context, sceneID uuid.UUID) (dialog.SceneInfo, error) {
	var info dialog.SceneInfo
	var limitType pgtype.Text
	var limitValue pgtype.Int4

	err := p.pool.QueryRow(ctx,
		`SELECT id, free_dialog_enabled, dialog_limit_type, dialog_limit_value FROM scenes WHERE id = $1`, sceneID,
	).Scan(&info.ID, &info.FreeDialogEnabled, &limitType, &limitValue)
	if err != nil {
		return dialog.SceneInfo{}, err
	}
	if limitType.Valid {
		info.DialogLimitType = &limitType.String
	}
	if limitValue.Valid {
		info.DialogLimitValue = &limitValue.Int32
	}
	return info, nil
}

type stubPublisher struct{}

func (stubPublisher) Publish(context.Context, string, any) error { return nil }

type stubNotifier struct{}

func (stubNotifier) SendToUser(string, []byte) error { return nil }

func TestDialogIntegration_MessageLimitEndsSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	authRepo := auth.NewPostgresRepository(pool)
	player, err := authRepo.CreateUser(ctx, "dialog-player@example.com", "hash")
	require.NoError(t, err)

	walletRepo := wallet.NewPostgresRepository(pool)
	walletService := wallet.NewService(walletRepo)
	_, err = walletService.Grant(ctx, player.ID, 1000)
	require.NoError(t, err)

	storyID := uuid.New()
	sceneID := uuid.New()
	characterID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO stories (id, title, is_published) VALUES ($1, $2, true)`, storyID, "Dialog Story")
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO scenes (id, story_id, order_index, dialogue_script, free_dialog_enabled, dialog_limit_type, dialog_limit_value)
		 VALUES ($1,$2,0,'[]',true,'messages',2)`, sceneID, storyID)
	require.NoError(t, err)

	repo := dialog.NewPostgresRepository(pool)
	svc := dialog.NewService(repo, poolSceneProvider{pool}, walletService, stubPublisher{}, stubNotifier{}, 20, nil)

	session, err := svc.StartSession(ctx, player.ID, characterID, sceneID)
	require.NoError(t, err)
	assert.Equal(t, dialog.StatusActive, session.Status)
	assert.Equal(t, dialog.LimitTypeMessages, session.LimitType)
	require.NotNil(t, session.LimitValue)
	assert.EqualValues(t, 2, *session.LimitValue)

	msg1, remaining1, err := svc.SendMessage(ctx, player.ID, session.ID, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", msg1.Text)
	require.NotNil(t, remaining1)
	assert.EqualValues(t, 1, *remaining1)

	msg2, remaining2, err := svc.SendMessage(ctx, player.ID, session.ID, "second message")
	require.NoError(t, err)
	assert.Equal(t, "second message", msg2.Text)
	require.NotNil(t, remaining2)
	assert.EqualValues(t, 0, *remaining2)

	// The second message exhausted the limit, so the session should now be
	// ended and a third message must be rejected.
	_, _, err = svc.SendMessage(ctx, player.ID, session.ID, "third message")
	require.Error(t, err)

	ended, err := svc.GetSession(ctx, player.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, dialog.StatusEnded, ended.Status)
	require.NotNil(t, ended.EndReason)
	assert.Equal(t, dialog.EndReasonMessageLimit, *ended.EndReason)

	// Two user messages at 20 diamonds each should have been debited.
	balance, err := walletService.Balance(ctx, player.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1000-40, balance)
}

func TestDialogIntegration_UserEndsSessionExplicitly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	authRepo := auth.NewPostgresRepository(pool)
	player, err := authRepo.CreateUser(ctx, "dialog-quitter@example.com", "hash")
	require.NoError(t, err)

	walletRepo := wallet.NewPostgresRepository(pool)
	walletService := wallet.NewService(walletRepo)
	_, err = walletService.Grant(ctx, player.ID, 100)
	require.NoError(t, err)

	storyID := uuid.New()
	sceneID := uuid.New()
	characterID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO stories (id, title, is_published) VALUES ($1, $2, true)`, storyID, "Dialog Story 2")
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO scenes (id, story_id, order_index, dialogue_script, free_dialog_enabled) VALUES ($1,$2,0,'[]',true)`,
		sceneID, storyID)
	require.NoError(t, err)

	repo := dialog.NewPostgresRepository(pool)
	svc := dialog.NewService(repo, poolSceneProvider{pool}, walletService, stubPublisher{}, stubNotifier{}, 20, nil)

	session, err := svc.StartSession(ctx, player.ID, characterID, sceneID)
	require.NoError(t, err)
	assert.Equal(t, dialog.LimitTypeNone, session.LimitType)

	ended, err := svc.EndSession(ctx, player.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, dialog.StatusEnded, ended.Status)
	require.NotNil(t, ended.EndReason)
	assert.Equal(t, dialog.EndReasonUser, *ended.EndReason)

	_, err = svc.EndSession(ctx, player.ID, session.ID)
	require.Error(t, err)
}
