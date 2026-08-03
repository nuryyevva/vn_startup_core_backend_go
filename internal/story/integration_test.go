//go:build integration

package story_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/internal/auth"
	"vn_startup_core_backend_go/internal/story"
	"vn_startup_core_backend_go/internal/testutil"
)

type stubWallet struct{}

func (stubWallet) Debit(context.Context, uuid.UUID, int32, string) error { return nil }

type stubPublisher struct{}

func (stubPublisher) Publish(context.Context, string, any) error { return nil }

func TestStoryIntegration_ProgressAndChoice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	authRepo := auth.NewPostgresRepository(pool)
	player, err := authRepo.CreateUser(ctx, "story-player@example.com", "hash")
	require.NoError(t, err)

	storyID := uuid.New()
	scene1ID := uuid.New()
	scene2ID := uuid.New()
	choiceID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO stories (id, title, is_published) VALUES ($1, $2, true)`, storyID, "Test Story")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO scenes (id, story_id, order_index, dialogue_script) VALUES ($1,$2,0,'[]')`, scene1ID, storyID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO scenes (id, story_id, order_index, dialogue_script) VALUES ($1,$2,1,'[]')`, scene2ID, storyID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO choices (id, scene_id, text, is_paid, cost_diamonds, next_scene_id) VALUES ($1,$2,'Go on',false,0,$3)`,
		choiceID, scene1ID, scene2ID)
	require.NoError(t, err)

	repo := story.NewPostgresRepository(pool)
	svc := story.NewService(repo, stubWallet{}, stubPublisher{}, nil)

	stories, err := svc.ListPublishedStories(ctx)
	require.NoError(t, err)
	require.Len(t, stories, 1)
	assert.Equal(t, storyID, stories[0].ID)

	progress, err := svc.GetOrCreateProgress(ctx, player.ID, storyID)
	require.NoError(t, err)
	assert.Equal(t, scene1ID, progress.CurrentSceneID)

	// A second call must return the same row, not create a duplicate one.
	progressAgain, err := svc.GetOrCreateProgress(ctx, player.ID, storyID)
	require.NoError(t, err)
	assert.Equal(t, progress.CurrentSceneID, progressAgain.CurrentSceneID)

	updated, err := svc.SubmitChoice(ctx, player.ID, scene1ID, choiceID)
	require.NoError(t, err)
	assert.Equal(t, scene2ID, updated.CurrentSceneID)
	assert.Contains(t, updated.ChoicesMade, choiceID)
}

func TestStoryIntegration_PaidChoiceChargesWallet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	authRepo := auth.NewPostgresRepository(pool)
	player, err := authRepo.CreateUser(ctx, "story-payer@example.com", "hash")
	require.NoError(t, err)

	storyID := uuid.New()
	sceneID := uuid.New()
	choiceID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO stories (id, title, is_published) VALUES ($1, $2, true)`, storyID, "Paid Story")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO scenes (id, story_id, order_index, dialogue_script) VALUES ($1,$2,0,'[]')`, sceneID, storyID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO choices (id, scene_id, text, is_paid, cost_diamonds) VALUES ($1,$2,'Buy hint',true,50)`, choiceID, sceneID)
	require.NoError(t, err)

	debited := false
	wallet := walletSpy{debit: func(userID uuid.UUID, amount int32, reason string) error {
		debited = true
		assert.Equal(t, player.ID, userID)
		assert.Equal(t, int32(50), amount)
		assert.Equal(t, "choice_purchase", reason)
		return nil
	}}

	repo := story.NewPostgresRepository(pool)
	svc := story.NewService(repo, wallet, stubPublisher{}, nil)

	_, err = svc.SubmitChoice(ctx, player.ID, sceneID, choiceID)
	require.NoError(t, err)
	assert.True(t, debited)
}

type walletSpy struct {
	debit func(userID uuid.UUID, amount int32, reason string) error
}

func (w walletSpy) Debit(_ context.Context, userID uuid.UUID, amount int32, reason string) error {
	return w.debit(userID, amount, reason)
}
