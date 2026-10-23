package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFogOverBlackwood_GraphIsValid(t *testing.T) {
	s, err := fogOverBlackwoodStory()
	require.NoError(t, err)

	sceneByID := map[uuid.UUID]scene{}
	last := s.scenes[0]
	for _, sc := range s.scenes {
		sceneByID[sc.id] = sc
		if sc.orderIndex > last.orderIndex {
			last = sc
		}
	}
	require.Len(t, sceneByID, len(s.scenes), "scene IDs must be unique")

	choiceIDs := map[uuid.UUID]bool{}
	choicesByScene := map[uuid.UUID][]choice{}
	for _, c := range s.choices {
		require.False(t, choiceIDs[c.id], "duplicate choice id %s", c.id)
		choiceIDs[c.id] = true
		require.NotNil(t, c.nextSceneID)
		_, ok := sceneByID[*c.nextSceneID]
		require.True(t, ok, "choice %q leads nowhere", c.text)
		choicesByScene[c.sceneID] = append(choicesByScene[c.sceneID], c)
	}

	// GET /scenes/:id lists choices ORDER BY id — that must match the
	// authoring order, which is the order they were appended in.
	for sceneID, cs := range choicesByScene {
		sorted := append([]choice{}, cs...)
		sort.Slice(sorted, func(i, j int) bool { return bytes.Compare(sorted[i].id[:], sorted[j].id[:]) < 0 })
		for i := range cs {
			assert.Equal(t, cs[i].text, sorted[i].text, "choice order in scene %s", sceneID)
		}
	}

	assert.Empty(t, choicesByScene[last.id], "the last scene (by order_index) ends the story")

	// Every scene can reach the ending, so no branch strands the player.
	reachesEnd := map[uuid.UUID]bool{last.id: true}
	for changed := true; changed; {
		changed = false
		for _, c := range s.choices {
			if reachesEnd[*c.nextSceneID] && !reachesEnd[c.sceneID] {
				reachesEnd[c.sceneID] = true
				changed = true
			}
		}
	}
	for _, sc := range s.scenes {
		assert.True(t, reachesEnd[sc.id], "scene at order %d can't reach the ending", sc.orderIndex)
	}

	chapters, titled := 0, 0
	for _, sc := range s.scenes {
		if sc.chapterTitle != nil {
			chapters++
		}
		if sc.title != nil {
			titled++
		}
		var lines []scriptLine
		require.NoError(t, json.Unmarshal([]byte(sc.dialogueScript), &lines))
		for _, l := range lines {
			assert.NotEmpty(t, l.Text)
			for _, cond := range l.When {
				for _, id := range cond.Among {
					assert.True(t, choiceIDs[id], "condition refers to unknown choice %s", id)
				}
			}
		}
	}
	assert.Equal(t, 1, chapters, "the episode is a single chapter")
	assert.Equal(t, 8, titled, "one titled section per scene of the script")
	assert.NotNil(t, s.scenes[0].chapterTitle, "the chapter starts at the first scene")
}

func TestFogOverBlackwood_TalkBeats(t *testing.T) {
	s, err := fogOverBlackwoodStory()
	require.NoError(t, err)

	talkers := map[uuid.UUID]int{}
	for _, sc := range s.scenes {
		if !sc.freeDialogEnabled {
			assert.Nil(t, sc.characterID, "a character without free dialog would show a bare silhouette")
			continue
		}
		require.NotNil(t, sc.characterID)
		// No length limit: each message costs diamonds instead.
		require.NotNil(t, sc.dialogLimitType)
		assert.Equal(t, "none", *sc.dialogLimitType)
		assert.Nil(t, sc.dialogLimitValue)
		assert.NotNil(t, sc.characterSpriteURL, "talk beat needs a sprite so the client doesn't fall back to a silhouette")
		require.NotNil(t, sc.characterName, "talk beat needs the character's name: a pure hub has no line to take it from")
		assert.NotEmpty(t, *sc.characterName)
		talkers[*sc.characterID]++
	}

	// Every character the AI module has a card for is reachable; the lists of
	// scenes live in vn_startup_ai/characters/<name>/card.yaml (talk_scenes).
	for name, want := range map[string]struct {
		id    uuid.UUID
		beats int
	}{
		"Адриан": {bwAdrianID, 2}, "Дэмиан": {bwDamianID, 1}, "Люси": {bwLucyID, 1},
		"Ной": {bwNoahID, 2}, "Морган": {bwMorganID, 1},
	} {
		assert.Equal(t, want.beats, talkers[want.id], "%s: number of talk beats", name)
	}
	assert.Len(t, talkers, 5)
}

func TestFogOverBlackwood_IDsAreStable(t *testing.T) {
	a, err := fogOverBlackwoodStory()
	require.NoError(t, err)
	b, err := fogOverBlackwoodStory()
	require.NoError(t, err)

	require.Equal(t, len(a.scenes), len(b.scenes))
	for i := range a.scenes {
		assert.Equal(t, a.scenes[i].id, b.scenes[i].id)
	}
	for i := range a.choices {
		assert.Equal(t, a.choices[i].id, b.choices[i].id)
	}
}

func TestFogOverBlackwood_SpriteFraming(t *testing.T) {
	s, err := fogOverBlackwoodStory()
	require.NoError(t, err)

	g := newStoryGraph(fogOverBlackwoodStoryID)
	byKey := map[string]scene{}
	for _, sc := range s.scenes {
		byKey[sc.id.String()] = sc
	}
	opening := func(key string) string { return string(byKey[g.sceneID(key).String()].spriteFraming) }

	assert.Equal(t, "full", opening("s1.arrival"), "outdoors: the whole figure")
	assert.Equal(t, "waist", opening("s2.meet"), "cafeteria table: conversation, cropped at the waist")
	assert.Equal(t, "waist", opening("s4.watch"), "the corridor: the usual conversation shot")
	assert.Equal(t, "waist", opening("s4.talk"), "talking face to face in the same corridor")
	// s6.road opens on the porch (the reaction to the porch choice) and then
	// cuts to the car.
	var road []scriptLine
	require.NoError(t, json.Unmarshal([]byte(byKey[g.sceneID("s6.road").String()].dialogueScript), &road))
	cut := slices.IndexFunc(road, func(l scriptLine) bool { return l.BackgroundURL == bwCarInterior })
	require.NotEqual(t, -1, cut)
	assert.Equal(t, "waist", string(road[cut].SpriteFraming), "inside the car")

	for _, sc := range s.scenes {
		var lines []scriptLine
		require.NoError(t, json.Unmarshal([]byte(sc.dialogueScript), &lines))
		for _, l := range lines {
			if l.BackgroundURL != "" {
				assert.NotEmpty(t, l.SpriteFraming, "a background switch must set its shot (%q)", l.Text)
			}
		}
	}
}
