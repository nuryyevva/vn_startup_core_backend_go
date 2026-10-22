// Command seed inserts sample stories — with scenes and choices — into the
// database: two short ones adapted from public-domain fairy tales, plus the
// original branching story "Туман над Блэквудом" (see
// fog_over_blackwood.go), so every read/write endpoint in the API (GET
// /stories, GET /stories/:id/progress, GET /scenes/:id, POST
// /scenes/:id/choice, and by extension the wallet and dialog endpoints) can
// be exercised manually without a content-authoring pipeline. There is no admin panel at this
// stage of the project, so this is the only way to get playable content
// into a fresh database.
//
// It is idempotent (every row uses a fixed UUID and INSERT ... ON CONFLICT
// DO UPDATE), so it's safe — and expected — to re-run after editing the
// content below; existing rows are updated in place rather than skipped,
// which is what makes it useful for iterating on art/audio URLs, and rows
// that disappeared from the content are pruned (see story.prune). It is not
// part of golang-migrate's schema migrations — it seeds data, not schema,
// and must never run against a production database.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/config"
	domain "vn_startup_core_backend_go/internal/story"
	"vn_startup_core_backend_go/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed: fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	exportDir := flag.String("export", "", "write the content bundle for the client's offline build to this directory instead of seeding the database")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Server.Env)
	ctx := context.Background()

	stories, err := sampleStories()
	if err != nil {
		return fmt.Errorf("build sample stories: %w", err)
	}

	// -export writes the content as a bundle for the client's offline
	// build instead of seeding the database (see export.go).
	if *exportDir != "" {
		if err := exportContent(ctx, stories, cfg.Media.Dir, *exportDir); err != nil {
			return fmt.Errorf("export content: %w", err)
		}
		log.Info("seed: exported content", slog.String("dir", *exportDir), slog.Int("stories", len(stories)))
		return nil
	}

	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	for _, s := range stories {
		if err := s.insert(ctx, pool); err != nil {
			return fmt.Errorf("seed story %q: %w", s.title, err)
		}
		log.Info("seed: inserted story",
			slog.String("title", s.title),
			slog.String("story_id", s.id.String()),
			slog.Int("scenes", len(s.scenes)),
		)
	}

	log.Info("seed: done")
	return nil
}

type story struct {
	id          uuid.UUID
	title       string
	description string
	coverURL    string
	genre       string
	status      string
	// playerTheme is the stories.player_theme JSON object ("" = default
	// look), e.g. `{"preset":"mist"}` — see README "Тема плеера".
	playerTheme string
	comingSoon  bool // listed but not playable yet (stories.coming_soon)
	scenes      []scene
	choices     []choice
}

type scene struct {
	id                 uuid.UUID
	orderIndex         int
	backgroundURL      string
	characterID        *uuid.UUID
	characterSpriteURL *string
	spriteFraming      domain.SpriteFraming // "" = the full figure
	backgroundMusicURL *string
	dialogueScript     string // JSON array literal
	freeDialogEnabled  bool
	dialogLimitType    *string // "time" | "messages" | nil
	dialogLimitValue   *int32
	unlockCostDiamonds *int32
	chapterTitle       *string // non-nil only on the scene that starts a chapter
	title              *string // non-nil only on scenes that start a section of a chapter
}

type choice struct {
	id           uuid.UUID
	sceneID      uuid.UUID
	text         string
	isPaid       bool
	costDiamonds int32
	nextSceneID  *uuid.UUID
	onceKey      *string // see migration 0017_choices_once_key
}

// framing is the scene's opening shot, defaulting to the full figure.
func (sc scene) framing() domain.SpriteFraming {
	if sc.spriteFraming == "" {
		return domain.SpriteFramingFull
	}
	return sc.spriteFraming
}

func strPtr(s string) *string        { return &s }
func i32Ptr(v int32) *int32          { return &v }
func uuidPtr(u uuid.UUID) *uuid.UUID { return &u }

func (s story) insert(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Every insert below is a real upsert (ON CONFLICT DO UPDATE), not just
	// ON CONFLICT DO NOTHING — this command is meant to be re-run as sample
	// content evolves (e.g. adding art/audio URLs), not only to be safe
	// against re-running with unchanged data.
	var playerTheme []byte // nil -> SQL NULL
	if s.playerTheme != "" {
		if !json.Valid([]byte(s.playerTheme)) {
			return fmt.Errorf("player theme is not valid JSON: %s", s.playerTheme)
		}
		playerTheme = []byte(s.playerTheme)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO stories (id, title, description, cover_url, genre, status, player_theme, coming_soon, is_published)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		 ON CONFLICT (id) DO UPDATE SET
		     title = EXCLUDED.title, description = EXCLUDED.description, cover_url = EXCLUDED.cover_url,
		     genre = EXCLUDED.genre, status = EXCLUDED.status, player_theme = EXCLUDED.player_theme,
		     coming_soon = EXCLUDED.coming_soon`,
		s.id, s.title, s.description, s.coverURL, s.genre, s.status, playerTheme, s.comingSoon,
	); err != nil {
		return fmt.Errorf("insert story: %w", err)
	}

	for _, sc := range s.scenes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO scenes (id, story_id, order_index, background_url, character_id, character_sprite_url, background_music_url, dialogue_script, free_dialog_enabled, dialog_limit_type, dialog_limit_value, unlock_cost_diamonds, title, chapter_title, sprite_framing)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			 ON CONFLICT (id) DO UPDATE SET
			     order_index = EXCLUDED.order_index, background_url = EXCLUDED.background_url,
			     character_id = EXCLUDED.character_id, character_sprite_url = EXCLUDED.character_sprite_url,
			     background_music_url = EXCLUDED.background_music_url, dialogue_script = EXCLUDED.dialogue_script,
			     free_dialog_enabled = EXCLUDED.free_dialog_enabled, dialog_limit_type = EXCLUDED.dialog_limit_type,
			     dialog_limit_value = EXCLUDED.dialog_limit_value, unlock_cost_diamonds = EXCLUDED.unlock_cost_diamonds,
			     title = EXCLUDED.title, chapter_title = EXCLUDED.chapter_title,
			     sprite_framing = EXCLUDED.sprite_framing`,
			sc.id, s.id, sc.orderIndex, sc.backgroundURL, sc.characterID, sc.characterSpriteURL, sc.backgroundMusicURL,
			[]byte(sc.dialogueScript), sc.freeDialogEnabled, sc.dialogLimitType, sc.dialogLimitValue, sc.unlockCostDiamonds,
			sc.title, sc.chapterTitle, sc.framing(),
		); err != nil {
			return fmt.Errorf("insert scene %s: %w", sc.id, err)
		}
	}

	for _, c := range s.choices {
		if _, err := tx.Exec(ctx,
			`INSERT INTO choices (id, scene_id, text, is_paid, cost_diamonds, next_scene_id, once_key)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (id) DO UPDATE SET
			     text = EXCLUDED.text, is_paid = EXCLUDED.is_paid, cost_diamonds = EXCLUDED.cost_diamonds,
			     next_scene_id = EXCLUDED.next_scene_id, once_key = EXCLUDED.once_key`,
			c.id, c.sceneID, c.text, c.isPaid, c.costDiamonds, c.nextSceneID, c.onceKey,
		); err != nil {
			return fmt.Errorf("insert choice %s: %w", c.id, err)
		}
	}

	if err := s.prune(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// prune removes this story's scenes and choices that are no longer part of
// the seed content, so editing a story (renaming, reordering or dropping a
// branch) doesn't leave stale choices showing up next to the current ones.
// Players whose progress points at a removed scene are moved back to the
// story's first scene — current_scene_id has no ON DELETE action, and
// restarting is the only safe guess once their position no longer exists.
func (s story) prune(ctx context.Context, tx pgx.Tx) error {
	if len(s.scenes) == 0 {
		return nil
	}

	sceneIDs := make([]string, 0, len(s.scenes))
	first := s.scenes[0]
	for _, sc := range s.scenes {
		sceneIDs = append(sceneIDs, sc.id.String())
		if sc.orderIndex < first.orderIndex {
			first = sc
		}
	}
	choiceIDs := make([]string, 0, len(s.choices))
	for _, c := range s.choices {
		choiceIDs = append(choiceIDs, c.id.String())
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM choices c USING scenes sc
		 WHERE c.scene_id = sc.id AND sc.story_id = $1 AND c.id <> ALL($2::uuid[])`,
		s.id, choiceIDs,
	); err != nil {
		return fmt.Errorf("prune choices: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE player_progress SET current_scene_id = $2
		 WHERE story_id = $1 AND current_scene_id <> ALL($3::uuid[])`,
		s.id, first.id, sceneIDs,
	); err != nil {
		return fmt.Errorf("reset progress on pruned scenes: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM scenes WHERE story_id = $1 AND id <> ALL($2::uuid[])`,
		s.id, sceneIDs,
	); err != nil {
		return fmt.Errorf("prune scenes: %w", err)
	}
	return nil
}

// Fixed IDs so re-running this command upserts the same rows (ON CONFLICT
// (id) DO UPDATE above) instead of accumulating duplicates.
var (
	// Story 1: Little Red Riding Hood (Charles Perrault / Brothers Grimm —
	// public domain fairy tale).
	redRidingHoodStoryID  = uuid.MustParse("11111111-1111-1111-1111-111111111101")
	redRidingHoodScene1ID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	redRidingHoodScene2ID = uuid.MustParse("11111111-1111-1111-1111-111111111112")
	redRidingHoodScene3ID = uuid.MustParse("11111111-1111-1111-1111-111111111113")
	redRidingHoodWolfID   = uuid.MustParse("11111111-1111-1111-1111-1111111111aa")

	redRidingHoodChoice1Free = uuid.MustParse("11111111-1111-1111-1111-111111111121")
	redRidingHoodChoice1Paid = uuid.MustParse("11111111-1111-1111-1111-111111111122")
	redRidingHoodChoice2Next = uuid.MustParse("11111111-1111-1111-1111-111111111123")

	// Story 2: Alice's Adventures in Wonderland, opening chapter (Lewis
	// Carroll, 1865 — public domain).
	aliceStoryID  = uuid.MustParse("22222222-2222-2222-2222-222222222201")
	aliceScene1ID = uuid.MustParse("22222222-2222-2222-2222-222222222211")
	aliceScene2ID = uuid.MustParse("22222222-2222-2222-2222-222222222212")
	aliceScene3ID = uuid.MustParse("22222222-2222-2222-2222-222222222213")
	aliceRabbitID = uuid.MustParse("22222222-2222-2222-2222-2222222222aa")

	aliceChoice1Free = uuid.MustParse("22222222-2222-2222-2222-222222222221")
	aliceChoice1Paid = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	aliceChoice2Next = uuid.MustParse("22222222-2222-2222-2222-222222222223")
)

// Placeholder art/audio for the two sample stories, so the scene player
// actually has something to render instead of a gray silhouette on a black
// screen — there's no content-authoring pipeline yet (see the file's top
// comment), so these are simply real URLs to public-domain/openly-licensed
// media on Wikimedia Commons, linked via its stable Special:FilePath
// redirect. When real story assets exist, replace these with whatever
// storage scheme is chosen then (object storage + CDN, per
// infrastructure-sizing notes) — nothing downstream cares that these are
// hotlinked Commons URLs today.
const (
	// Little Red Riding Hood — illustrations by Walter Crane (1875) and
	// Gustave Doré (public domain); "Knocking on wood or door.ogg" and
	// "Satie Gymnopedie No. 3" recording are both public domain (pdsounds.org
	// via Wikimedia user Fæ; Satie died 1925, composition and this specific
	// recording are both PD).
	ridingHoodHomeBackground    = "https://upload.wikimedia.org/wikipedia/commons/7/71/WalterCrane%2CLittleRedRidingHood-1.png"
	ridingHoodForestBackground  = "https://upload.wikimedia.org/wikipedia/commons/4/44/WalterCrane%2CLittle_Red_Riding_Hood-3.png"
	ridingHoodCottageBackground = "https://upload.wikimedia.org/wikipedia/commons/4/4d/WalterCrane%2CLittle_Red_Riding_Hood-5.png"
	ridingHoodWolfSprite        = "https://upload.wikimedia.org/wikipedia/commons/b/bf/GustaveDore_She_was_astonished_to_see_how_her_grandmother_looked.1.jpg"
	ridingHoodDoorKnockSfx      = "https://upload.wikimedia.org/wikipedia/commons/1/1c/Knocking_on_wood_or_door.ogg"
	ridingHoodBackgroundMusic   = "https://upload.wikimedia.org/wikipedia/commons/e/e1/Satie_Gymnopedie_No._3_for_piano_solo_01.wav"
	// The book's own cover plate — not the wolf/bed engraving used above,
	// so the library grid doesn't repeat the same image as both cover and
	// sprite.
	ridingHoodCover = "https://thumb.wikimedia.org/wikipedia/commons/thumb/e/e6/WalterCrane%2CLittleRedRidingHood-0.png/500px-WalterCrane%2CLittleRedRidingHood-0.png"

	// Alice's Adventures in Wonderland — John Tenniel's original 1865
	// engravings (public domain, author died 1914); "Clock ticking.ogg" is
	// public domain (pdsounds.org via Fæ); the Debussy recording is CC BY
	// 3.0, performed by Laurens Goedhart — attribution required if reused
	// elsewhere.
	aliceRiverbankBackground  = "https://upload.wikimedia.org/wikipedia/commons/8/83/Alice-white-rabbit.jpg"
	aliceRabbitHoleBackground = "https://upload.wikimedia.org/wikipedia/commons/3/3a/Alice_par_John_Tenniel_03.png"
	aliceDoorsHallBackground  = "https://upload.wikimedia.org/wikipedia/commons/6/63/Alice_par_John_Tenniel_04.png"
	aliceRabbitSprite         = "https://upload.wikimedia.org/wikipedia/commons/e/e4/White_Rabbit_Illustration.png"
	aliceClockTickSfx         = "https://upload.wikimedia.org/wikipedia/commons/5/56/Clock_ticking.ogg"
	// CC BY 3.0 — Laurens Goedhart, https://soundcloud.com/laurensgoedhart/claude-debussys-clair-de-lune
	aliceBackgroundMusic = "https://upload.wikimedia.org/wikipedia/commons/b/be/Clair_de_lune_%28Claude_Debussy%29_Suite_bergamasque.ogg"
	aliceCover           = "https://encrypted-tbn1.gstatic.com/images?q=tbn:ANd9GcSCRURJV79F1fADzxBT6MRq7RrfchiXfQeFIeLZ3OC81boJOWpb"
)

func sampleStories() ([]story, error) {
	blackwood, err := fogOverBlackwoodStory()
	if err != nil {
		return nil, fmt.Errorf("fog over blackwood: %w", err)
	}
	return []story{redRidingHoodStory(), aliceStory(), blackwood}, nil
}

func redRidingHoodStory() story {
	return story{
		id:          redRidingHoodStoryID,
		title:       "Красная Шапочка",
		description: "Классическая сказка о девочке в красной шапочке, тропинке через лес и Волке, который выдаёт себя за бабушку.",
		coverURL:    ridingHoodCover,
		genre:       "fantasy",
		status:      "completed",
		playerTheme: `{"preset":"storybook"}`,
		comingSoon:  true,
		scenes: []scene{
			{
				id:                 redRidingHoodScene1ID,
				orderIndex:         0,
				backgroundURL:      ridingHoodHomeBackground,
				backgroundMusicURL: strPtr(ridingHoodBackgroundMusic),
				dialogueScript:     `[{"speaker":"narrator","text":"Жила-была на свете маленькая девочка, и мать любила её без памяти, а бабушка ещё больше."},{"speaker":"Мама","text":"Отнеси бабушке пирожок и горшочек масла, да не задерживайся в лесу."}]`,
				freeDialogEnabled:  false,
			},
			{
				id:                 redRidingHoodScene2ID,
				orderIndex:         1,
				backgroundURL:      ridingHoodForestBackground,
				characterID:        uuidPtr(redRidingHoodWolfID),
				characterSpriteURL: strPtr(ridingHoodWolfSprite),
				backgroundMusicURL: strPtr(ridingHoodBackgroundMusic),
				dialogueScript:     `[{"speaker":"narrator","text":"На тропинке девочку встретил Волк. Он был не прочь узнать, куда она идёт."},{"speaker":"Волк","text":"Куда путь держишь, дитя, в такой ранний час?"}]`,
				freeDialogEnabled:  true,
			},
			{
				id:                 redRidingHoodScene3ID,
				orderIndex:         2,
				backgroundURL:      ridingHoodCottageBackground,
				backgroundMusicURL: strPtr(ridingHoodBackgroundMusic),
				dialogueScript:     `[{"speaker":"narrator","text":"Девочка постучала в дверь бабушкиного домика.","sfx_url":"` + ridingHoodDoorKnockSfx + `"},{"speaker":"narrator","text":"\"Кто там?\" — раздался странно низкий голос."}]`,
				freeDialogEnabled:  false,
				unlockCostDiamonds: i32Ptr(25),
			},
		},
		choices: []choice{
			{
				id:           redRidingHoodChoice1Free,
				sceneID:      redRidingHoodScene1ID,
				text:         "Пойти к бабушке напрямик через лес",
				isPaid:       false,
				costDiamonds: 0,
				nextSceneID:  uuidPtr(redRidingHoodScene2ID),
			},
			{
				id:           redRidingHoodChoice1Paid,
				sceneID:      redRidingHoodScene1ID,
				text:         "Заплатить дровосеку, чтобы он проводил тебя в обход опасной тропы (10 алмазов)",
				isPaid:       true,
				costDiamonds: 10,
				nextSceneID:  uuidPtr(redRidingHoodScene2ID),
			},
			{
				id:           redRidingHoodChoice2Next,
				sceneID:      redRidingHoodScene2ID,
				text:         "Попрощаться с Волком и поспешить к домику бабушки",
				isPaid:       false,
				costDiamonds: 0,
				nextSceneID:  uuidPtr(redRidingHoodScene3ID),
			},
		},
	}
}

func aliceStory() story {
	return story{
		id:          aliceStoryID,
		title:       "Алиса в Стране чудес",
		description: "Скучающая на берегу реки Алиса замечает спешащего Белого Кролика с карманными часами — и следует за ним в кроличью нору.",
		coverURL:    aliceCover,
		genre:       "adventure",
		status:      "ongoing",
		// A preset plus one overridden token: gold accents on the
		// wonderland palette.
		playerTheme: `{"preset":"wonderland","accent_color":"#E8C36A"}`,
		comingSoon:  true,
		scenes: []scene{
			{
				id:                 aliceScene1ID,
				orderIndex:         0,
				backgroundURL:      aliceRiverbankBackground,
				backgroundMusicURL: strPtr(aliceBackgroundMusic),
				dialogueScript:     `[{"speaker":"narrator","text":"Алисе наскучило сидеть без дела на берегу реки рядом с сестрой, у которой не было ни картинок, ни разговоров в книге."},{"speaker":"narrator","text":"Вдруг мимо пробежал Белый Кролик с розовыми глазами."},{"speaker":"Белый Кролик","text":"Ах, боже мой, боже мой! Я опаздываю!","sfx_url":"` + aliceClockTickSfx + `"}]`,
				freeDialogEnabled:  false,
			},
			{
				id:                 aliceScene2ID,
				orderIndex:         1,
				backgroundURL:      aliceRabbitHoleBackground,
				characterID:        uuidPtr(aliceRabbitID),
				characterSpriteURL: strPtr(aliceRabbitSprite),
				backgroundMusicURL: strPtr(aliceBackgroundMusic),
				dialogueScript:     `[{"speaker":"narrator","text":"Алиса нырнула вслед за Кроликом в нору, даже не подумав, как же она будет выбираться обратно."},{"speaker":"narrator","text":"Нора шла прямо, как туннель, а потом внезапно обрывалась вниз — и падение оказалось на удивление медленным."}]`,
				freeDialogEnabled:  true,
			},
			{
				id:                 aliceScene3ID,
				orderIndex:         2,
				backgroundURL:      aliceDoorsHallBackground,
				backgroundMusicURL: strPtr(aliceBackgroundMusic),
				dialogueScript:     `[{"speaker":"narrator","text":"Алиса оказалась в длинном низком зале, вдоль которого стоял ряд запертых дверей, а на стеклянном столике лежал крошечный золотой ключик."}]`,
				freeDialogEnabled:  false,
				unlockCostDiamonds: i32Ptr(15),
			},
		},
		choices: []choice{
			{
				id:           aliceChoice1Free,
				sceneID:      aliceScene1ID,
				text:         "Побежать за Белым Кроликом к кроличьей норе",
				isPaid:       false,
				costDiamonds: 0,
				nextSceneID:  uuidPtr(aliceScene2ID),
			},
			{
				id:           aliceChoice1Paid,
				sceneID:      aliceScene1ID,
				text:         "Достать волшебное увеличительное стекло, чтобы разглядеть нору получше (15 алмазов)",
				isPaid:       true,
				costDiamonds: 15,
				nextSceneID:  uuidPtr(aliceScene2ID),
			},
			{
				id:           aliceChoice2Next,
				sceneID:      aliceScene2ID,
				text:         "Взять золотой ключик и открыть маленькую дверцу",
				isPaid:       false,
				costDiamonds: 0,
				nextSceneID:  uuidPtr(aliceScene3ID),
			},
		},
	}
}
