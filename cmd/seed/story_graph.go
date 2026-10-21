package main

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	domain "vn_startup_core_backend_go/internal/story"
)

// This file is a small authoring helper for branching stories. The data
// model has no notion of a "fork inside a scene": a scene is a linear
// dialogue script followed by choices, each pointing at the next scene. So a
// branching chapter is authored as a graph of short scenes ("beats") linked
// by choices, and storyGraph turns that graph into the `story` rows the seed
// inserts.
//
// Two conventions keep the graph small:
//
//   - A choice can carry the lines that react to it. They're prepended to
//     the target beat's script with a `when` condition (see lineCondition),
//     so several choices can converge on one beat without a separate beat
//     per reaction. This also makes "explore the room" hubs a single beat
//     that links back to itself: each item is a choice whose reaction lines
//     play on return.
//   - Lines are the dialogue_script JSON objects the client already renders
//     ({"speaker","text","sfx_url"}), extended with sticky per-line
//     background/sprite/framing switches and the `when` condition — see
//     scriptLine.
//
// Sprite framing (how close the camera is on the character, see
// domain.SpriteFraming) mostly follows the place: storyGraph.framingFor maps
// a background to its usual shot, and build applies it both to the scene
// (scenes.sprite_framing) and to every line that switches the background.
// A beat can override its opening shot with graphBeat.framed, a line with
// scriptLine.framed — e.g. pulling back to the full figure for a moment
// that is about the whole body (someone stepping back, rising).

// scriptLine is one entry of scenes.dialogue_script.
type scriptLine struct {
	// Speaker is a character name, or one of two sentinels: "narrator"
	// (scene description) and "thought" (the heroine's inner voice).
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
	// SpriteURL switches the displayed character from this line on; an
	// empty string hides the character. Nil keeps whatever was shown.
	SpriteURL *string `json:"sprite_url,omitempty"`
	// BackgroundURL switches the background from this line on.
	BackgroundURL string `json:"background_url,omitempty"`
	// SpriteFraming switches the shot from this line on. Like the other
	// switches it sticks until another line changes it, so a line that pulls
	// back for one moment needs a later line that moves in again.
	SpriteFraming domain.SpriteFraming `json:"sprite_framing,omitempty"`
	SfxURL        string               `json:"sfx_url,omitempty"`
	// When lists conditions that must all hold for the line to be shown.
	When []lineCondition `json:"when,omitempty"`
}

// lineCondition is evaluated by the client against the player's
// progress.choices_made: take the most recent choice in choices_made that
// belongs to Among. `chosen` holds if it's one of Chosen; `not_chosen`
// holds if there is none or it isn't one of NotChosen. Looking only at the
// most recent one (rather than "ever chosen") keeps replays correct, since
// choices_made is append-only across playthroughs.
type lineCondition struct {
	Chosen    []uuid.UUID `json:"chosen,omitempty"`
	NotChosen []uuid.UUID `json:"not_chosen,omitempty"`
	Among     []uuid.UUID `json:"among"`
}

const (
	speakerNarrator = "narrator"
	speakerThought  = "thought"
)

func narr(text string) scriptLine    { return scriptLine{Speaker: speakerNarrator, Text: text} }
func thought(text string) scriptLine { return scriptLine{Speaker: speakerThought, Text: text} }

// portrait is a speaking character with the sprite shown while they talk.
// An empty sprite hides the character layer (e.g. someone only heard in a
// vision).
type portrait struct {
	name   string
	sprite string
}

func (p portrait) say(text string) scriptLine {
	return scriptLine{Speaker: p.name, Text: text}.show(p)
}

func (l scriptLine) show(p portrait) scriptLine {
	sprite := p.sprite
	l.SpriteURL = &sprite
	return l
}

func (l scriptLine) hide() scriptLine {
	empty := ""
	l.SpriteURL = &empty
	return l
}

func (l scriptLine) bg(url string) scriptLine {
	l.BackgroundURL = url
	return l
}

// framed switches the sprite framing from this line on (see SpriteFraming).
func (l scriptLine) framed(f domain.SpriteFraming) scriptLine {
	l.SpriteFraming = f
	return l
}

func (l scriptLine) when(c lineCondition) scriptLine {
	l.When = append(append([]lineCondition{}, l.When...), c)
	return l
}

// chosen builds a condition that holds when the latest of among (plus ids
// themselves) is one of ids.
func chosen(ids []uuid.UUID, among ...uuid.UUID) lineCondition {
	return lineCondition{Chosen: ids, Among: union(ids, among)}
}

// notChosen is the complement of chosen: it also holds when none of among
// has been chosen yet.
func notChosen(ids []uuid.UUID, among ...uuid.UUID) lineCondition {
	return lineCondition{NotChosen: ids, Among: union(ids, among)}
}

func union(a, b []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(a)+len(b))
	out := make([]uuid.UUID, 0, len(a)+len(b))
	for _, id := range append(append([]uuid.UUID{}, a...), b...) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

type storyGraph struct {
	// ns seeds every scene/choice UUID, which are derived from authoring
	// keys rather than counters so adding or reordering beats doesn't shift
	// the IDs of unrelated rows (and with them players' saved progress).
	ns uuid.UUID
	// music loops under every beat (scenes.background_music_url); the
	// client keeps it playing across beats as long as the URL is the same.
	music string
	// musicFor, when set, picks the track for a beat by its key and overrides
	// music for beats it returns a non-empty URL for. Keep it stable within a
	// stretch of the story: the client restarts the track whenever the URL
	// changes between beats.
	musicFor func(beatKey string) string
	// framingFor, when set, gives the usual sprite framing for a background
	// URL (empty = no opinion, i.e. the full figure). See the file comment.
	framingFor func(backgroundURL string) domain.SpriteFraming
	beats      []*graphBeat
	byKey      map[string]*graphBeat
	links      []*graphLink
	nextPos    map[string]int
}

type graphBeat struct {
	key     string
	chapter *string
	title   *string
	bg      string
	framing domain.SpriteFraming // overrides framingFor(bg) when set
	lines   []scriptLine
	talk    *talkTo
}

// talkTo turns a beat into one where the player can start a free dialog with
// an AI-driven character (scenes.character_id + free_dialog_enabled). The
// character's persona and memories live in the AI module's characters/
// folder under the same ID.
type talkTo struct {
	characterID uuid.UUID
	sprite      string // shown whenever no line has chosen a sprite yet
	messages    int32  // free-dialog message limit
}

type graphLink struct {
	ids   []uuid.UUID
	from  []string
	text  string
	to    string
	lines []scriptLine
	once  bool
}

func newStoryGraph(ns uuid.UUID) *storyGraph {
	return &storyGraph{ns: ns, byKey: map[string]*graphBeat{}, nextPos: map[string]int{}}
}

// beat declares a scene. Declaration order is the scene's order_index, so
// declare beats in reading order and the final one last — the backend
// considers a story finished once progress reaches the highest index.
func (g *storyGraph) beat(key, bg string, lines ...scriptLine) *graphBeat {
	b := &graphBeat{key: key, bg: bg, lines: lines}
	g.beats = append(g.beats, b)
	g.byKey[key] = b
	return b
}

// talksWith lets the player start a free dialog with the character in this
// beat. Pick beats where the character is on screen and the player can
// linger (hubs): the "talk" button sits next to the beat's choices.
func (b *graphBeat) talksWith(characterID uuid.UUID, sprite string, messages int32) *graphBeat {
	b.talk = &talkTo{characterID: characterID, sprite: sprite, messages: messages}
	return b
}

// framed sets the shot the beat opens with, overriding the one its
// background would get from storyGraph.framingFor.
func (b *graphBeat) framed(f domain.SpriteFraming) *graphBeat {
	b.framing = f
	return b
}

// startsChapter marks the beat as the first of a chapter
// (scenes.chapter_title).
func (b *graphBeat) startsChapter(title string) *graphBeat {
	b.chapter = &title
	return b
}

// titled marks the beat as the start of a named section of its chapter
// (scenes.title) — what the chapter list shows when the chapter is expanded.
func (b *graphBeat) titled(title string) *graphBeat {
	b.title = &title
	return b
}

// link adds a choice labelled text to every beat in from, all leading to
// the beat `to`. lines (optional) are played at the start of `to` only when
// it's reached through one of these choices. Returns the created choice IDs
// in the order of from, for use in later line conditions.
func (g *storyGraph) link(from []string, text, to string, lines ...scriptLine) []uuid.UUID {
	l := &graphLink{from: from, text: text, to: to, lines: lines}
	for _, f := range from {
		pos := g.nextPos[f]
		g.nextPos[f] = pos + 1
		l.ids = append(l.ids, g.choiceID(f, pos))
	}
	g.links = append(g.links, l)
	return l.ids
}

// explore is link for an item of an exploration hub: the choice disappears
// once taken (choices.once_key), so the list shrinks as the player works
// through it instead of offering what they've already seen. Every `from`
// beat shares one key — typically the hub itself and the beat leading into
// it. A beat must keep at least one plain link, or it could run out of
// choices (build checks this).
func (g *storyGraph) explore(from []string, text, to string, lines ...scriptLine) []uuid.UUID {
	ids := g.link(from, text, to, lines...)
	g.links[len(g.links)-1].once = true
	return ids
}

func (g *storyGraph) sceneID(key string) uuid.UUID {
	return uuid.NewSHA1(g.ns, []byte("scene:"+key))
}

// choiceID is name-based like sceneID, except its first byte is the
// choice's position within its scene: choices are listed ORDER BY id, so
// this keeps them in authoring order.
func (g *storyGraph) choiceID(fromKey string, pos int) uuid.UUID {
	id := uuid.NewSHA1(g.ns, []byte(fmt.Sprintf("choice:%s#%d", fromKey, pos)))
	id[0] = byte(pos)
	return id
}

// build validates the graph and produces the scenes and choices to insert.
func (g *storyGraph) build() ([]scene, []choice, error) {
	if len(g.beats) == 0 {
		return nil, nil, fmt.Errorf("story has no beats")
	}

	inbound := map[string][]uuid.UUID{}
	outbound := map[string]int{}
	onward := map[string]int{} // outbound choices that never disappear
	for _, l := range g.links {
		if _, ok := g.byKey[l.to]; !ok {
			return nil, nil, fmt.Errorf("choice %q leads to unknown beat %q", l.text, l.to)
		}
		for _, f := range l.from {
			if _, ok := g.byKey[f]; !ok {
				return nil, nil, fmt.Errorf("choice %q starts from unknown beat %q", l.text, f)
			}
			outbound[f]++
			if !l.once {
				onward[f]++
			}
		}
		inbound[l.to] = append(inbound[l.to], l.ids...)
	}

	last := g.beats[len(g.beats)-1]
	for _, b := range g.beats {
		if b != last && outbound[b.key] == 0 {
			return nil, nil, fmt.Errorf("beat %q is a dead end: only the final beat may have no choices", b.key)
		}
		if b.key != g.beats[0].key && len(inbound[b.key]) == 0 {
			return nil, nil, fmt.Errorf("beat %q is unreachable", b.key)
		}
		if outbound[b.key] > 0 && onward[b.key] == 0 {
			return nil, nil, fmt.Errorf("beat %q has only one-time choices and could run out of them", b.key)
		}
		if outbound[b.key] > 255 {
			return nil, nil, fmt.Errorf("beat %q has more than 255 choices", b.key)
		}
	}
	if outbound[last.key] != 0 {
		return nil, nil, fmt.Errorf("final beat %q must not have choices", last.key)
	}

	scripts := map[string][]scriptLine{}
	for _, l := range g.links {
		for _, line := range l.lines {
			scripts[l.to] = append(scripts[l.to], line.when(chosen(l.ids, inbound[l.to]...)))
		}
	}

	scenes := make([]scene, 0, len(g.beats))
	for i, b := range g.beats {
		script := g.applyFraming(append(scripts[b.key], b.lines...))
		framing := b.framing
		if framing == "" {
			framing = g.framingOf(b.bg)
		}
		if !framing.Valid() {
			return nil, nil, fmt.Errorf("beat %q has unknown sprite framing %q", b.key, framing)
		}
		for _, line := range script {
			if line.SpriteFraming != "" && !line.SpriteFraming.Valid() {
				return nil, nil, fmt.Errorf("a line of beat %q has unknown sprite framing %q", b.key, line.SpriteFraming)
			}
		}
		raw, err := json.Marshal(script)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal script of beat %q: %w", b.key, err)
		}
		track := g.music
		if g.musicFor != nil {
			if url := g.musicFor(b.key); url != "" {
				track = url
			}
		}
		var music *string
		if track != "" {
			music = strPtr(track)
		}
		sc := scene{
			id:                 g.sceneID(b.key),
			orderIndex:         i,
			backgroundURL:      b.bg,
			spriteFraming:      framing,
			backgroundMusicURL: music,
			dialogueScript:     string(raw),
			chapterTitle:       b.chapter,
			title:              b.title,
		}
		if t := b.talk; t != nil {
			sc.characterID = uuidPtr(t.characterID)
			if t.sprite != "" {
				sc.characterSpriteURL = strPtr(t.sprite)
			}
			sc.freeDialogEnabled = true
			sc.dialogLimitType = strPtr("messages")
			sc.dialogLimitValue = i32Ptr(t.messages)
		}
		scenes = append(scenes, sc)
	}

	choices := make([]choice, 0, len(g.links))
	for _, l := range g.links {
		var onceKey *string
		if l.once {
			onceKey = strPtr(l.ids[0].String())
		}
		for i, f := range l.from {
			choices = append(choices, choice{
				id:          l.ids[i],
				sceneID:     g.sceneID(f),
				text:        l.text,
				nextSceneID: uuidPtr(g.sceneID(l.to)),
				onceKey:     onceKey,
			})
		}
	}

	return scenes, choices, nil
}

// framingOf is the usual shot for a background (full figure unless
// framingFor says otherwise).
func (g *storyGraph) framingOf(backgroundURL string) domain.SpriteFraming {
	if g.framingFor != nil {
		if f := g.framingFor(backgroundURL); f != "" {
			return f
		}
	}
	return domain.SpriteFramingFull
}

// applyFraming gives every line that switches the background the shot that
// background usually gets, unless the line picks one itself. Works on a
// copy: a link's lines are shared by every beat it leads to.
func (g *storyGraph) applyFraming(script []scriptLine) []scriptLine {
	out := make([]scriptLine, len(script))
	for i, line := range script {
		if line.BackgroundURL != "" && line.SpriteFraming == "" {
			line.SpriteFraming = g.framingOf(line.BackgroundURL)
		}
		out[i] = line
	}
	return out
}
