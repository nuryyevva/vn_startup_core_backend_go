package achievement

// ActivityStats is the aggregate, cross-module activity snapshot each
// achievement's Check function is evaluated against. It's assembled by an
// ActivityProvider that reads from the story, wallet, dialog, and stats
// modules — the achievement module itself never touches their tables
// directly.
type ActivityStats struct {
	FinishedStories    int32
	UnlockedScenes     int32
	DialogMessagesSent int64
	DiamondsSpent      int64
	DayStreak          int32
}

// Definition is one entry in the hardcoded achievement catalog. There is no
// content-authoring UI for achievements (see project README's "known
// limitations" for admin tooling in general) so, for now, the catalog is
// Go code rather than a database table — editing it means a deploy, same as
// editing the genre list in internal/story.
type Definition struct {
	ID          string
	Title       string
	Description string
	Check       func(ActivityStats) bool
}

// Catalog is every achievement in the game, evaluated in this order. IDs are
// permanent once shipped — user_achievements rows reference them by ID, so
// renaming an ID here effectively resets that achievement for everyone.
var Catalog = []Definition{
	{
		ID:          "first_story_finished",
		Title:       "Первая история",
		Description: "Заверши свою первую историю",
		Check:       func(s ActivityStats) bool { return s.FinishedStories >= 1 },
	},
	{
		ID:          "story_collector",
		Title:       "Коллекционер историй",
		Description: "Заверши 3 истории",
		Check:       func(s ActivityStats) bool { return s.FinishedStories >= 3 },
	},
	{
		ID:          "bookworm",
		Title:       "Книжный червь",
		Description: "Заверши 10 историй",
		Check:       func(s ActivityStats) bool { return s.FinishedStories >= 10 },
	},
	{
		ID:          "first_words",
		Title:       "Первые слова",
		Description: "Отправь своё первое сообщение в свободном диалоге",
		Check:       func(s ActivityStats) bool { return s.DialogMessagesSent >= 1 },
	},
	{
		ID:          "chatterbox",
		Title:       "Болтушка",
		Description: "Отправь 50 сообщений в свободном диалоге",
		Check:       func(s ActivityStats) bool { return s.DialogMessagesSent >= 50 },
	},
	{
		ID:          "deep_talker",
		Title:       "Глубокий разговор",
		Description: "Отправь 200 сообщений в свободном диалоге",
		Check:       func(s ActivityStats) bool { return s.DialogMessagesSent >= 200 },
	},
	{
		ID:          "generous",
		Title:       "Первая трата",
		Description: "Потрать свои первые алмазы",
		Check:       func(s ActivityStats) bool { return s.DiamondsSpent >= 1 },
	},
	{
		ID:          "big_spender",
		Title:       "Щедрая душа",
		Description: "Потрать 500 алмазов",
		Check:       func(s ActivityStats) bool { return s.DiamondsSpent >= 500 },
	},
	{
		ID:          "explorer",
		Title:       "Исследователь",
		Description: "Разблокируй 5 платных глав",
		Check:       func(s ActivityStats) bool { return s.UnlockedScenes >= 5 },
	},
	{
		ID:          "week_streak",
		Title:       "Неделя подряд",
		Description: "Читай истории 7 дней подряд",
		Check:       func(s ActivityStats) bool { return s.DayStreak >= 7 },
	},
	{
		ID:          "month_streak",
		Title:       "Месяц подряд",
		Description: "Читай истории 30 дней подряд",
		Check:       func(s ActivityStats) bool { return s.DayStreak >= 30 },
	},
}
