package main

import (
	"bytes"
	"encoding/json"
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
