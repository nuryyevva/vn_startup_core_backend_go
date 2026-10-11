package story

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// One-time choices.
//
// A scene can loop back to itself — "look around the room": each thing to
// inspect is a choice that returns to the same list. A choice with a
// OnceKey is offered only once per visit to its scene, so the list shrinks
// as the player explores it instead of letting them go round in circles.
//
// "Per visit" is the part that needs care, because progress.choices_made
// is an append-only log that survives replays: "ever chosen" would leave a
// hub permanently empty on the second playthrough. A visit starts with the
// last choice the player made on an earlier scene (a lower order_index):
// everything after it was chosen on the scene itself or on later scenes
// that loop back to it, i.e. without leaving. A choice is used up if it,
// or another choice with the same OnceKey, was made at or after that
// point. Matching by key rather than by ID is what lets the scene that
// leads into a hub offer the same options: picking one there both starts
// the visit and is its first use.

// usedOneTimeChoices returns which of choices (all belonging to scene) the
// player has already used during their current visit to it.
func (s *Service) usedOneTimeChoices(ctx context.Context, userID uuid.UUID, scene Scene, choices []Choice) (map[uuid.UUID]bool, error) {
	var keys []string
	for _, c := range choices {
		if c.OnceKey != nil {
			keys = append(keys, *c.OnceKey)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}

	progress, err := s.repo.GetProgress(ctx, userID, scene.StoryID)
	if err != nil {
		if errors.Is(err, ErrProgressNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if len(progress.ChoicesMade) == 0 {
		return nil, nil
	}

	madeEarlier, err := s.repo.ListChoiceIDsFromEarlierScenes(ctx, scene, progress.ChoicesMade)
	if err != nil {
		return nil, err
	}
	keyByChoice, err := s.repo.ListChoiceOnceKeys(ctx, scene.StoryID, keys)
	if err != nil {
		return nil, err
	}

	visitStart := 0
	for i := len(progress.ChoicesMade) - 1; i >= 0; i-- {
		if madeEarlier[progress.ChoicesMade[i]] {
			visitStart = i
			break
		}
	}

	usedKeys := map[string]bool{}
	for _, made := range progress.ChoicesMade[visitStart:] {
		if key, ok := keyByChoice[made]; ok {
			usedKeys[key] = true
		}
	}

	used := map[uuid.UUID]bool{}
	for _, c := range choices {
		if c.OnceKey != nil && usedKeys[*c.OnceKey] {
			used[c.ID] = true
		}
	}
	return used, nil
}

// withoutUsedChoices drops the one-time choices the player has already
// used. A scene must never be left with nothing to choose (the story
// couldn't continue), so if every choice turns out to be used — which
// correctly authored content rules out by keeping a way onward that isn't
// one-time — the full list is returned instead.
func withoutUsedChoices(choices []Choice, used map[uuid.UUID]bool) []Choice {
	if len(used) == 0 {
		return choices
	}
	available := make([]Choice, 0, len(choices))
	for _, c := range choices {
		if !used[c.ID] {
			available = append(available, c)
		}
	}
	if len(available) == 0 {
		return choices
	}
	return available
}
