package story

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/pkg/apperr"
)

// hubFixture is an exploration hub: two things to look at (one-time) and a
// way onward, entered through a choice made on an earlier scene.
type hubFixture struct {
	repo              *mockRepository
	svc               *Service
	userID            uuid.UUID
	scene             Scene
	entry             uuid.UUID
	coat, photo, exit Choice
}

func newHubFixture(choicesMade func(f *hubFixture) []uuid.UUID) *hubFixture {
	f := &hubFixture{repo: new(mockRepository), userID: uuid.New(), entry: uuid.New()}
	f.svc = NewService(f.repo, new(mockWallet), nil, nil)
	f.scene = Scene{ID: uuid.New(), StoryID: uuid.New(), OrderIndex: 3}

	coatKey, photoKey := "coat", "photo"
	f.coat = Choice{ID: uuid.New(), SceneID: f.scene.ID, NextSceneID: &f.scene.ID, OnceKey: &coatKey}
	f.photo = Choice{ID: uuid.New(), SceneID: f.scene.ID, NextSceneID: &f.scene.ID, OnceKey: &photoKey}
	f.exit = Choice{ID: uuid.New(), SceneID: f.scene.ID, NextSceneID: &f.scene.ID}

	made := choicesMade(f)
	f.repo.On("GetScene", mock.Anything, f.scene.ID).Return(f.scene, nil)
	f.repo.On("ListChoices", mock.Anything, f.scene.ID).Return([]Choice{f.coat, f.photo, f.exit}, nil)
	f.repo.On("GetProgress", mock.Anything, f.userID, f.scene.StoryID).
		Return(Progress{UserID: f.userID, StoryID: f.scene.StoryID, CurrentSceneID: f.scene.ID, ChoicesMade: made}, nil)
	f.repo.On("ListChoiceIDsFromEarlierScenes", mock.Anything, f.scene, made).
		Return(map[uuid.UUID]bool{f.entry: true}, nil)
	f.repo.On("ListChoiceOnceKeys", mock.Anything, f.scene.StoryID, []string{"coat", "photo"}).
		Return(map[uuid.UUID]string{f.coat.ID: "coat", f.photo.ID: "photo"}, nil)
	return f
}

func (f *hubFixture) offered(t *testing.T) []uuid.UUID {
	t.Helper()
	_, choices, err := f.svc.GetSceneForPlayer(context.Background(), f.userID, f.scene.ID)
	require.NoError(t, err)
	ids := make([]uuid.UUID, 0, len(choices))
	for _, c := range choices {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestService_GetSceneForPlayer_HidesOneTimeChoicesAlreadyUsedThisVisit(t *testing.T) {
	f := newHubFixture(func(f *hubFixture) []uuid.UUID { return []uuid.UUID{f.entry, f.coat.ID} })

	assert.Equal(t, []uuid.UUID{f.photo.ID, f.exit.ID}, f.offered(t))
}

func TestService_GetSceneForPlayer_OffersOneTimeChoicesAgainOnANewVisit(t *testing.T) {
	// A full earlier playthrough of the hub, then the scene is entered
	// again: nothing from the first visit counts.
	f := newHubFixture(func(f *hubFixture) []uuid.UUID {
		return []uuid.UUID{f.entry, f.coat.ID, f.photo.ID, f.exit.ID, f.entry}
	})

	assert.Equal(t, []uuid.UUID{f.coat.ID, f.photo.ID, f.exit.ID}, f.offered(t))
}

func TestService_GetSceneForPlayer_LeavesOnlyTheWayOnwardOnceEverythingIsExplored(t *testing.T) {
	f := newHubFixture(func(f *hubFixture) []uuid.UUID { return []uuid.UUID{f.entry, f.photo.ID, f.coat.ID} })

	assert.Equal(t, []uuid.UUID{f.exit.ID}, f.offered(t))
}

func TestService_SubmitChoice_RejectsAUsedOneTimeChoice(t *testing.T) {
	f := newHubFixture(func(f *hubFixture) []uuid.UUID { return []uuid.UUID{f.entry, f.coat.ID} })
	f.repo.On("GetChoice", mock.Anything, f.coat.ID).Return(f.coat, nil)

	_, err := f.svc.SubmitChoice(context.Background(), f.userID, f.scene.ID, f.coat.ID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "choice_already_made", appErr.Code)
	f.repo.AssertNotCalled(t, "UpdateProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestWithoutUsedChoices_NeverLeavesASceneWithNothingToChoose(t *testing.T) {
	a, b := Choice{ID: uuid.New()}, Choice{ID: uuid.New()}

	got := withoutUsedChoices([]Choice{a, b}, map[uuid.UUID]bool{a.ID: true, b.ID: true})

	assert.Equal(t, []Choice{a, b}, got)
}
