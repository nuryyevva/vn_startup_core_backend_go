package main

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "vn_startup_core_backend_go/internal/story"
)

// framingGraph is a three-beat graph whose framingFor makes "room" a
// close-up background and leaves everything else to the default.
func framingGraph() *storyGraph {
	g := newStoryGraph(uuid.MustParse("44444444-4444-4444-4444-444444444401"))
	g.framingFor = func(bg string) domain.SpriteFraming {
		if bg == "room" {
			return domain.SpriteFramingKnees
		}
		return ""
	}
	return g
}

// scriptOf decodes a built scene's dialogue_script.
func scriptOf(t *testing.T, sc scene) []scriptLine {
	t.Helper()
	var lines []scriptLine
	require.NoError(t, json.Unmarshal([]byte(sc.dialogueScript), &lines))
	return lines
}

func TestStoryGraph_SceneFramingFollowsBackground(t *testing.T) {
	g := framingGraph()
	g.beat("street", "street", narr("a"))
	g.beat("room", "room", narr("b"))
	g.beat("hall", "street", narr("c")).framed(domain.SpriteFramingKnees)
	g.link(from("street"), "in", "room")
	g.link(from("room"), "on", "hall")

	scenes, _, err := g.build()
	require.NoError(t, err)

	assert.Equal(t, domain.SpriteFramingFull, scenes[0].spriteFraming, "no opinion from framingFor means the full figure")
	assert.Equal(t, domain.SpriteFramingKnees, scenes[1].spriteFraming, "taken from the background")
	assert.Equal(t, domain.SpriteFramingKnees, scenes[2].spriteFraming, "a beat's own framing wins over its background's")
}

func TestStoryGraph_BackgroundSwitchCarriesItsFraming(t *testing.T) {
	g := framingGraph()
	g.beat("street", "street",
		narr("outside"),
		narr("goes in").bg("room"),
		narr("looks out").bg("street"),
		narr("steps back in, but stays wide").bg("room").framed(domain.SpriteFramingFull),
		narr("leans in").framed(domain.SpriteFramingKnees),
	)

	scenes, _, err := g.build()
	require.NoError(t, err)

	got := make([]domain.SpriteFraming, 0, 5)
	for _, l := range scriptOf(t, scenes[0]) {
		got = append(got, l.SpriteFraming)
	}
	assert.Equal(t, []domain.SpriteFraming{
		"",                        // no switch: keeps the scene's shot
		domain.SpriteFramingKnees, // the room's usual shot
		domain.SpriteFramingFull,  // the street's (framingFor has no opinion)
		domain.SpriteFramingFull,  // the line's own choice beats the room's
		domain.SpriteFramingKnees, // a plain override
	}, got)
}

func TestStoryGraph_FramingDoesNotLeakBetweenTargets(t *testing.T) {
	// One link's reaction lines are copied into its target beat; resolving
	// framing there must not write back into the shared slice.
	g := framingGraph()
	lines := []scriptLine{narr("cut to the room").bg("room")}
	g.beat("start", "street")
	g.beat("end", "street")
	g.link(from("start"), "go", "end", lines...)

	_, _, err := g.build()
	require.NoError(t, err)
	assert.Empty(t, lines[0].SpriteFraming)
}

func TestStoryGraph_RejectsUnknownFraming(t *testing.T) {
	g := framingGraph()
	g.beat("only", "street", narr("x").framed("waist"))
	_, _, err := g.build()
	require.Error(t, err)

	g = framingGraph()
	g.beat("only", "street").framed("waist")
	_, _, err = g.build()
	require.Error(t, err)
}
