package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Export writes the seed content as a self-contained bundle for the
// client's offline build (`-export <dir>`): <dir>/content.json plus every
// media file it refers to under <dir>/media. Nothing touches the database —
// the bundle is built from the same story definitions the seed inserts, so
// the two can't drift.
//
// content.json mirrors the API's story/scene/choice responses, so the
// client parses it with the models it already has. Media URLs in it are all
// root-relative ("/media/..."): files served by the API are copied from
// MEDIA_DIR, hotlinked ones (the two fairy-tale stories) are downloaded
// into media/remote.

type exportedContent struct {
	Stories []exportedStory `json:"stories"`
}

type exportedStory struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	CoverURL    string          `json:"cover_url"`
	Genre       string          `json:"genre"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	PlayerTheme json.RawMessage `json:"player_theme,omitempty"`
	Scenes      []exportedScene `json:"scenes"`
}

type exportedScene struct {
	ID                 string           `json:"id"`
	StoryID            string           `json:"story_id"`
	OrderIndex         int              `json:"order_index"`
	BackgroundURL      string           `json:"background_url,omitempty"`
	CharacterID        *string          `json:"character_id,omitempty"`
	DialogueScript     []map[string]any `json:"dialogue_script"`
	FreeDialogEnabled  bool             `json:"free_dialog_enabled"`
	DialogLimitType    *string          `json:"dialog_limit_type,omitempty"`
	DialogLimitValue   *int32           `json:"dialog_limit_value,omitempty"`
	UnlockCostDiamonds *int32           `json:"unlock_cost_diamonds,omitempty"`
	CharacterSpriteURL *string          `json:"character_sprite_url,omitempty"`
	BackgroundMusicURL *string          `json:"background_music_url,omitempty"`
	ChapterTitle       *string          `json:"chapter_title,omitempty"`
	Title              *string          `json:"title,omitempty"`
	Choices            []exportedChoice `json:"choices"`
}

type exportedChoice struct {
	ID           string  `json:"id"`
	Text         string  `json:"text"`
	IsPaid       bool    `json:"is_paid"`
	CostDiamonds int32   `json:"cost_diamonds"`
	NextSceneID  *string `json:"next_scene_id,omitempty"`
	OnceKey      *string `json:"once_key,omitempty"`
}

// scriptMediaKeys are the dialogue_script line fields that hold media URLs.
var scriptMediaKeys = []string{"background_url", "sprite_url", "sfx_url"}

type exporter struct {
	mediaDir string // where the API serves /media from
	outDir   string
	client   *http.Client
}

func exportContent(ctx context.Context, stories []story, mediaDir, outDir string) error {
	e := exporter{mediaDir: mediaDir, outDir: outDir, client: &http.Client{Timeout: 2 * time.Minute}}

	// The catalog lists newest first; give the stories distinct, stable
	// timestamps so the bundle keeps the order the seed inserts them in.
	epoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	content := exportedContent{}
	for i, s := range stories {
		out, err := e.story(ctx, s)
		if err != nil {
			return fmt.Errorf("story %q: %w", s.title, err)
		}
		out.CreatedAt = epoch.Add(time.Duration(i) * time.Hour)
		content.Stories = append(content.Stories, out)
	}

	raw, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "content.json"), raw, 0o644)
}

func (e exporter) story(ctx context.Context, s story) (exportedStory, error) {
	cover, err := e.media(ctx, s.coverURL)
	if err != nil {
		return exportedStory{}, err
	}
	out := exportedStory{
		ID:          s.id.String(),
		Title:       s.title,
		Description: s.description,
		CoverURL:    cover,
		Genre:       s.genre,
		Status:      s.status,
	}
	if s.playerTheme != "" {
		out.PlayerTheme = json.RawMessage(s.playerTheme)
	}

	choicesByScene := map[string][]choice{}
	for _, c := range s.choices {
		key := c.sceneID.String()
		choicesByScene[key] = append(choicesByScene[key], c)
	}

	scenes := append([]scene{}, s.scenes...)
	sort.Slice(scenes, func(i, j int) bool { return scenes[i].orderIndex < scenes[j].orderIndex })
	for _, sc := range scenes {
		exported, err := e.scene(ctx, s, sc, choicesByScene[sc.id.String()])
		if err != nil {
			return exportedStory{}, fmt.Errorf("scene %d: %w", sc.orderIndex, err)
		}
		out.Scenes = append(out.Scenes, exported)
	}
	return out, nil
}

func (e exporter) scene(ctx context.Context, s story, sc scene, choices []choice) (exportedScene, error) {
	var script []map[string]any
	if err := json.Unmarshal([]byte(sc.dialogueScript), &script); err != nil {
		return exportedScene{}, fmt.Errorf("parse dialogue script: %w", err)
	}
	for _, line := range script {
		for _, key := range scriptMediaKeys {
			// An empty sprite_url is the "hide the character" marker.
			if raw, ok := line[key].(string); ok && raw != "" {
				local, err := e.media(ctx, raw)
				if err != nil {
					return exportedScene{}, err
				}
				line[key] = local
			}
		}
	}

	background, err := e.media(ctx, sc.backgroundURL)
	if err != nil {
		return exportedScene{}, err
	}
	sprite, err := e.optionalMedia(ctx, sc.characterSpriteURL)
	if err != nil {
		return exportedScene{}, err
	}
	music, err := e.optionalMedia(ctx, sc.backgroundMusicURL)
	if err != nil {
		return exportedScene{}, err
	}

	out := exportedScene{
		ID:                 sc.id.String(),
		StoryID:            s.id.String(),
		OrderIndex:         sc.orderIndex,
		BackgroundURL:      background,
		DialogueScript:     script,
		FreeDialogEnabled:  sc.freeDialogEnabled,
		DialogLimitType:    sc.dialogLimitType,
		DialogLimitValue:   sc.dialogLimitValue,
		UnlockCostDiamonds: sc.unlockCostDiamonds,
		CharacterSpriteURL: sprite,
		BackgroundMusicURL: music,
		ChapterTitle:       sc.chapterTitle,
		Title:              sc.title,
		Choices:            []exportedChoice{},
	}
	if sc.characterID != nil {
		out.CharacterID = strPtr(sc.characterID.String())
	}

	// The API lists a scene's choices ORDER BY id; keep that order.
	sorted := append([]choice{}, choices...)
	sort.Slice(sorted, func(i, j int) bool { return bytes.Compare(sorted[i].id[:], sorted[j].id[:]) < 0 })
	for _, c := range sorted {
		ec := exportedChoice{
			ID:           c.id.String(),
			Text:         c.text,
			IsPaid:       c.isPaid,
			CostDiamonds: c.costDiamonds,
			OnceKey:      c.onceKey,
		}
		if c.nextSceneID != nil {
			ec.NextSceneID = strPtr(c.nextSceneID.String())
		}
		out.Choices = append(out.Choices, ec)
	}
	return out, nil
}

func (e exporter) optionalMedia(ctx context.Context, raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	local, err := e.media(ctx, *raw)
	if err != nil {
		return nil, err
	}
	return &local, nil
}

// media makes sure the file behind raw is in the bundle and returns its
// root-relative URL there.
func (e exporter) media(ctx context.Context, raw string) (string, error) {
	switch {
	case raw == "":
		return "", nil
	case strings.HasPrefix(raw, "/media/"):
		rel, err := url.PathUnescape(strings.TrimPrefix(raw, "/media/"))
		if err != nil {
			return "", fmt.Errorf("decode %s: %w", raw, err)
		}
		if err := copyFile(filepath.Join(e.mediaDir, filepath.FromSlash(rel)), e.bundlePath(raw)); err != nil {
			return "", fmt.Errorf("copy %s: %w", raw, err)
		}
		return raw, nil
	case strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "https://"):
		return e.download(ctx, raw)
	default:
		return "", fmt.Errorf("don't know how to bundle media URL %q", raw)
	}
}

func (e exporter) bundlePath(mediaURL string) string {
	return filepath.Join(e.outDir, filepath.FromSlash(strings.TrimPrefix(mediaURL, "/")))
}

// download fetches a hotlinked file into media/remote, named by a hash of
// its URL so re-exports reuse what's already there.
func (e exporter) download(ctx context.Context, raw string) (string, error) {
	sum := sha1.Sum([]byte(raw))
	name := hex.EncodeToString(sum[:8])

	// Reuse a previous download, whatever extension it ended up with.
	if matches, _ := filepath.Glob(e.bundlePath("/media/remote/" + name + ".*")); len(matches) > 0 {
		return "/media/remote/" + filepath.Base(matches[0]), nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	// Wikimedia rejects requests without a descriptive User-Agent.
	req.Header.Set("User-Agent", "vn-startup-seed-export/1.0 (content bundling for local development)")
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", raw, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: status %s", raw, resp.Status)
	}

	local := "/media/remote/" + name + mediaExtension(raw, resp.Header.Get("Content-Type"))
	dst := e.bundlePath(local)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(dst)
		return "", fmt.Errorf("download %s: %w", raw, err)
	}
	return local, f.Close()
}

// mediaExtension picks a file extension for a downloaded file: the URL's
// own if it has one, otherwise one matching the response's content type
// (some image hosts serve extensionless URLs).
func mediaExtension(raw, contentType string) string {
	if u, err := url.Parse(raw); err == nil {
		if ext := strings.ToLower(path.Ext(u.Path)); ext != "" && len(ext) <= 5 {
			return ext
		}
	}
	switch strings.TrimSpace(strings.Split(contentType, ";")[0]) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg", "application/ogg":
		return ".ogg"
	default:
		return ".jpg"
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
