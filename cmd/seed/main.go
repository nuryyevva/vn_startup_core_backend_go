// Command seed inserts a couple of sample stories — with scenes and
// choices — into the database, adapted from public-domain fairy tales, so
// every read/write endpoint in the API (GET /stories, GET
// /stories/:id/progress, GET /scenes/:id, POST /scenes/:id/choice, and by
// extension the wallet and dialog endpoints) can be exercised manually
// without a content-authoring pipeline. There is no admin panel at this
// stage of the project, so this is the only way to get playable content
// into a fresh database.
//
// It is idempotent (every row uses a fixed UUID and INSERT ... ON CONFLICT
// DO NOTHING), so it's safe to run more than once against the same
// database, and is not part of golang-migrate's schema migrations — it
// seeds data, not schema, and must never run against a production
// database.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/config"
	"vn_startup_core_backend_go/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed: fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Server.Env)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	for _, s := range sampleStories() {
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
	scenes      []scene
	choices     []choice
}

type scene struct {
	id                uuid.UUID
	orderIndex        int
	backgroundURL     string
	characterID       *uuid.UUID
	dialogueScript    string // JSON array literal
	freeDialogEnabled bool
	dialogLimitType   *string // "time" | "messages" | nil
	dialogLimitValue  *int32
}

type choice struct {
	id           uuid.UUID
	sceneID      uuid.UUID
	text         string
	isPaid       bool
	costDiamonds int32
	nextSceneID  *uuid.UUID
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

	if _, err := tx.Exec(ctx,
		`INSERT INTO stories (id, title, description, cover_url, is_published)
		 VALUES ($1, $2, $3, $4, true)
		 ON CONFLICT (id) DO NOTHING`,
		s.id, s.title, s.description, s.coverURL,
	); err != nil {
		return fmt.Errorf("insert story: %w", err)
	}

	for _, sc := range s.scenes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO scenes (id, story_id, order_index, background_url, character_id, dialogue_script, free_dialog_enabled, dialog_limit_type, dialog_limit_value)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (id) DO NOTHING`,
			sc.id, s.id, sc.orderIndex, sc.backgroundURL, sc.characterID,
			[]byte(sc.dialogueScript), sc.freeDialogEnabled, sc.dialogLimitType, sc.dialogLimitValue,
		); err != nil {
			return fmt.Errorf("insert scene %s: %w", sc.id, err)
		}
	}

	for _, c := range s.choices {
		if _, err := tx.Exec(ctx,
			`INSERT INTO choices (id, scene_id, text, is_paid, cost_diamonds, next_scene_id)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO NOTHING`,
			c.id, c.sceneID, c.text, c.isPaid, c.costDiamonds, c.nextSceneID,
		); err != nil {
			return fmt.Errorf("insert choice %s: %w", c.id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Fixed IDs so re-running this command is idempotent (ON CONFLICT (id) DO
// NOTHING above) instead of accumulating duplicate rows.
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

func sampleStories() []story {
	return []story{redRidingHoodStory(), aliceStory()}
}

func redRidingHoodStory() story {
	return story{
		id:          redRidingHoodStoryID,
		title:       "Красная Шапочка",
		description: "Классическая сказка о девочке в красной шапочке, тропинке через лес и Волке, который выдаёт себя за бабушку.",
		coverURL:    "https://upload.wikimedia.org/wikipedia/commons/thumb/6/64/Gustave_Dor%C3%A9_-_Le_Petit_Chaperon_rouge.jpg/400px-Gustave_Dor%C3%A9_-_Le_Petit_Chaperon_rouge.jpg",
		scenes: []scene{
			{
				id:                redRidingHoodScene1ID,
				orderIndex:        0,
				backgroundURL:     "https://example.com/backgrounds/riding-hood/home.jpg",
				dialogueScript:    `[{"speaker":"narrator","text":"Жила-была на свете маленькая девочка, и мать любила её без памяти, а бабушка ещё больше."},{"speaker":"Мама","text":"Отнеси бабушке пирожок и горшочек масла, да не задерживайся в лесу."}]`,
				freeDialogEnabled: false,
			},
			{
				id:                redRidingHoodScene2ID,
				orderIndex:        1,
				backgroundURL:     "https://example.com/backgrounds/riding-hood/forest.jpg",
				characterID:       uuidPtr(redRidingHoodWolfID),
				dialogueScript:    `[{"speaker":"narrator","text":"На тропинке девочку встретил Волк. Он был не прочь узнать, куда она идёт."},{"speaker":"Волк","text":"Куда путь держишь, дитя, в такой ранний час?"}]`,
				freeDialogEnabled: true,
				dialogLimitType:   strPtr("messages"),
				dialogLimitValue:  i32Ptr(5),
			},
			{
				id:                redRidingHoodScene3ID,
				orderIndex:        2,
				backgroundURL:     "https://example.com/backgrounds/riding-hood/cottage.jpg",
				dialogueScript:    `[{"speaker":"narrator","text":"Девочка постучала в дверь бабушкиного домика. \"Кто там?\" — раздался странно низкий голос."}]`,
				freeDialogEnabled: false,
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
		coverURL:    "https://upload.wikimedia.org/wikipedia/commons/thumb/8/85/Alice_par_John_Tenniel_02.png/400px-Alice_par_John_Tenniel_02.png",
		scenes: []scene{
			{
				id:                aliceScene1ID,
				orderIndex:        0,
				backgroundURL:     "https://example.com/backgrounds/alice/riverbank.jpg",
				dialogueScript:    `[{"speaker":"narrator","text":"Алисе наскучило сидеть без дела на берегу реки рядом с сестрой, у которой не было ни картинок, ни разговоров в книге."},{"speaker":"narrator","text":"Вдруг мимо пробежал Белый Кролик с розовыми глазами."},{"speaker":"Белый Кролик","text":"Ах, боже мой, боже мой! Я опаздываю!"}]`,
				freeDialogEnabled: false,
			},
			{
				id:                aliceScene2ID,
				orderIndex:        1,
				backgroundURL:     "https://example.com/backgrounds/alice/rabbit-hole.jpg",
				characterID:       uuidPtr(aliceRabbitID),
				dialogueScript:    `[{"speaker":"narrator","text":"Алиса нырнула вслед за Кроликом в нору, даже не подумав, как же она будет выбираться обратно."},{"speaker":"narrator","text":"Нора шла прямо, как туннель, а потом внезапно обрывалась вниз — и падение оказалось на удивление медленным."}]`,
				freeDialogEnabled: true,
				dialogLimitType:   strPtr("time"),
				dialogLimitValue:  i32Ptr(120),
			},
			{
				id:                aliceScene3ID,
				orderIndex:        2,
				backgroundURL:     "https://example.com/backgrounds/alice/garden-door.jpg",
				dialogueScript:    `[{"speaker":"narrator","text":"Алиса оказалась в длинном низком зале, вдоль которого стоял ряд запертых дверей, а на стеклянном столике лежал крошечный золотой ключик."}]`,
				freeDialogEnabled: false,
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
