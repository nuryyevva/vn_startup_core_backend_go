package main

import (
	"strings"

	"github.com/google/uuid"
)

// "Туман над Блэквудом", episode 1 "Чужая память" — an original branching
// mystery/romance (one chapter of 8 scenes, two endings), authored with
// storyGraph (see story_graph.go for the beat/choice conventions used below).
//
// Art lives in media/stories/fog-over-blackwood and is served by the API
// itself under /media (see config.MediaConfig); the URLs below are
// root-relative on purpose — the client resolves them against whatever API
// base URL it was built with.

var fogOverBlackwoodStoryID = uuid.MustParse("33333333-3333-3333-3333-333333333301")

// Characters the player can talk to freely. These IDs are the link to the AI
// module: its characters/<name>/card.yaml files carry the same `id`. Where each
// of them can be talked to is set with .talksWith(...) on the beats below.
// Only beats where it is natural to stay and chat for a while and the character
// is still present get it (s2.hall and s3.noah Noah, s2.lucy Lucy, s2.damian
// Damian, s4.watch and s6.road Adrian, s5.room Morgan); the button stays for the
// whole beat, so it never goes on a beat the character leaves. Evelyn is the
// player's own character and has no card.
var (
	bwAdrianID = uuid.MustParse("33333333-3333-3333-3333-3333333333a1")
	bwDamianID = uuid.MustParse("33333333-3333-3333-3333-3333333333a2")
	bwLucyID   = uuid.MustParse("33333333-3333-3333-3333-3333333333a3")
	bwNoahID   = uuid.MustParse("33333333-3333-3333-3333-3333333333a4")
	bwMorganID = uuid.MustParse("33333333-3333-3333-3333-3333333333a5")
)

// bwTalkMessages is the free-dialog message limit of every talk beat.
const bwTalkMessages = 8

const blackwoodMedia = "/media/stories/fog-over-blackwood/"

const (
	bwCover = blackwoodMedia + "cover.webp"

	// Soundtrack. Sources and licences: media/stories/fog-over-blackwood/CREDITS.md.
	bwTrackNarrative = blackwoodMedia + "audio/%D0%BF%D0%BE%D0%B2%D0%B5%D1%81%D1%82%D0%B2%D0%BE%D0%B2%D0%B0%D1%82%D0%B5%D0%BB%D1%8C%D0%BD%D0%B0%D1%8F.mp3"
	bwTrackTense     = blackwoodMedia + "audio/%D0%BD%D0%B0%D0%BF%D1%80%D1%8F%D0%B6%D0%B5%D0%BD%D0%BD%D0%B0%D1%8F_%D0%BA%D0%BE%D0%BD%D0%B5%D1%86_%D0%BC%D0%BE%D0%B6%D0%BD%D0%BE_%D0%BE%D0%B1%D1%80%D0%B5%D0%B7%D0%B0%D1%82%D1%8C.mp3"
	bwTrackCalm      = blackwoodMedia + "audio/%D1%80%D0%BE%D0%BC%D0%B0%D0%BD%D1%82%D0%B8%D1%87%D0%B5%D1%81%D0%BA%D0%B0%D1%8F%2C%20%D1%81%D0%BF%D0%BE%D0%BA%D0%BE%D0%B9%D0%BD%D0%B0%D1%8F.mp3"
	bwTrackSad       = blackwoodMedia + "audio/%D1%80%D0%BE%D0%BC%D0%B0%D0%BD%D1%82%D0%B8%D1%87%D0%B5%D1%81%D0%BA%D0%B0%D1%8F%2C%20%D1%87%D1%83%D1%82%D1%8C%20%D0%B3%D1%80%D1%83%D1%81%D1%82%D0%BD%D0%B0%D1%8F.mp3"
	bwTrackBackdrop  = blackwoodMedia + "audio/%D1%84%D0%BE%D0%BD.mp3"
	bwTrackAmbient   = blackwoodMedia + "audio/%D1%84%D0%BE%D0%BD%D0%BE%D0%B2%D0%B0%D1%8F.mp3"

	// Default track, used by any beat musicFor has no pick for.
	bwMusic = bwTrackBackdrop

	bwHouseExterior    = blackwoodMedia + "backgrounds/grandma_house_exterior.webp"
	bwKitchen          = blackwoodMedia + "backgrounds/grandma_kitchen.webp"
	bwCafeteria        = blackwoodMedia + "backgrounds/university_cafeteria.webp"
	bwParking          = blackwoodMedia + "backgrounds/university_parking.webp"
	bwCorridor         = blackwoodMedia + "backgrounds/old_building_corridor.webp"
	bwArchive          = blackwoodMedia + "backgrounds/university_archive.webp"
	bwForestRoad       = blackwoodMedia + "backgrounds/forest_road.webp"
	bwCarInterior      = blackwoodMedia + "backgrounds/car_interior.webp"
	bwForesterExterior = blackwoodMedia + "backgrounds/forester_house_exterior.webp"
	bwForesterInterior = blackwoodMedia + "backgrounds/forester_house_interior.webp"
)

var (
	evelyn        = portrait{"Эвелин", blackwoodMedia + "sprites/evelyn_neutral.webp"}
	evelynSad     = portrait{"Эвелин", blackwoodMedia + "sprites/evelyn_sad.webp"}
	evelynWorried = portrait{"Эвелин", blackwoodMedia + "sprites/evelyn_worried.webp"}
	adrian        = portrait{"Адриан", blackwoodMedia + "sprites/adrian.webp"}
	damian        = portrait{"Дэмиан", blackwoodMedia + "sprites/damian.webp"}
	lucy          = portrait{"Люси", blackwoodMedia + "sprites/lucy.webp"}
	noah          = portrait{"Ной", blackwoodMedia + "sprites/noah.webp"}
	morgan        = portrait{"Морган", blackwoodMedia + "sprites/morgan.webp"}
	taxiDriver    = portrait{"Водитель", blackwoodMedia + "sprites/taxi_driver.webp"}
	carDriver     = portrait{"Водитель", blackwoodMedia + "sprites/car_driver.webp"}
	// Only ever seen in visions/memories, through someone else's eyes — no
	// sprite on screen.
	grandma        = portrait{"Бабушка", ""}
	adrianInMemory = portrait{"Адриан", ""}
)

func from(keys ...string) []string { return keys }

// bwSectionTracks gives each of the story's eight sections (beat keys "s1.*" …
// "s8.*") a track by its mood: a switch of track happens only at a section
// border, since the client restarts the music whenever the URL changes.
var bwSectionTracks = map[string]string{
	"s1": bwTrackNarrative, // arrival at the grandmother's house
	"s2": bwTrackAmbient,   // university cafeteria, meeting the Vails
	"s3": bwTrackTense,     // the accident in the parking lot
	"s4": bwTrackBackdrop,  // old building corridor, Adrian, the key
	"s5": bwTrackNarrative, // archive, the journal
	"s6": bwTrackCalm,      // the road with Adrian, the forester's house
	"s7": bwTrackTense,     // searching the forester's house, the vision
	"s8": bwTrackSad,       // the choice and the ending
}

func bwSectionMusic() func(beatKey string) string {
	return func(beatKey string) string {
		section, _, _ := strings.Cut(beatKey, ".")
		return bwSectionTracks[section]
	}
}

func fogOverBlackwoodStory() (story, error) {
	g := newStoryGraph(fogOverBlackwoodStoryID)
	g.music = bwMusic
	g.musicFor = bwSectionMusic()

	sceneOne(g)
	sceneTwo(g)
	hasPhoto, noPhoto := sceneThree(g)
	sceneFour(g, hasPhoto, noPhoto)
	sceneFive(g)
	sceneSix(g)
	sceneSeven(g)
	sceneEight(g)

	scenes, choices, err := g.build()
	if err != nil {
		return story{}, err
	}
	return story{
		id:          fogOverBlackwoodStoryID,
		title:       "Туман над Блэквудом",
		description: "Эвелин Морроу приезжает в Блэквуд на похороны бабушки и находит ключ, который нужно отдать лично незнакомому профессору. Авария на парковке, юноша, оказавшийся рядом слишком быстро, и чужое воспоминание о ночи, когда умерла бабушка, — у озера Без Эха её ждут ответы, которых ей никто не хотел давать.",
		coverURL:    bwCover,
		genre:       "mystery",
		status:      "ongoing",
		playerTheme: `{"preset":"mist"}`,
		scenes:      scenes,
		choices:     choices,
	}, nil
}

// --- Сцена 1. Дом, который тебя помнит ---

func sceneOne(g *storyGraph) {
	g.beat("s1.arrival", bwHouseExterior,
		narr("Двухэтажный деревянный дом стоит в конце узкой улицы. За соседними участками ещё видны фонари, но здесь освещено только крыльцо.").hide(),
		narr("Сразу за оградой начинается лес."),
		narr("Краска на перилах потрескалась. Под навесом висят пустые цветочные горшки, которые бабушка каждую весну наполняла геранью."),
		narr("Один раскачивается от ветра и негромко постукивает о стену."),
		narr("Такси останавливается у калитки."),
		taxiDriver.say("Здесь?"),
		evelyn.say("Да. Спасибо."),
		narr("Он достаёт чемодан, но вместо того, чтобы сразу передать его, смотрит на окна.").show(taxiDriver),
		taxiDriver.say("Вы ей внучка?"),
		evelyn.say("Вы знали мою бабушку?"),
		taxiDriver.say("Здесь все всех знают."),
	).startsChapter("Чужая память").titled("Дом, который тебя помнит")

	g.link(from("s1.arrival"), "Расспросить о бабушке", "s1.inside",
		evelyn.say("Вы видели её незадолго до смерти?"),
		taxiDriver.say("Подвозил пару раз. Из университета."),
		evelynSad.say("Она выглядела больной?"),
		taxiDriver.say("Уставшей. Но сумку мне не разрешила нести. Сказала, у меня спина хуже её."),
		narr("Эвелин впервые за весь день улыбается.").show(evelyn),
		thought("Да. Это на неё похоже. Хотя бы что-то здесь пока похоже на правду."),
		narr("Водитель оставляет визитку на чемодане.").show(taxiDriver),
		taxiDriver.say("Если понадобится машина, звоните."),
		thought("Значит, она ездила в университет до самого конца.").show(evelyn),
	)
	g.link(from("s1.arrival"), "Спросить, почему он смотрит на дом", "s1.inside",
		evelynWorried.say("С домом что-то не так?"),
		taxiDriver.say("Нет, просто непривычно без света."),
		narr("Он кивает на кухонное окно."),
		taxiDriver.say("Она всегда оставляла там лампу."),
		thought("Я тоже это помню. Возвращаешься с улицы и ещё от калитки знаешь, что тебя ждут.").show(evelynSad),
	)
	g.link(from("s1.arrival"), "Завершить разговор", "s1.inside",
		evelyn.say("Спасибо, дальше я сама."),
		taxiDriver.say("Конечно. Примите мои соболезнования."),
		narr("Он помогает поставить чемодан под навес и уезжает."),
		thought("Я не могу ещё раз слушать, какой хорошей она была. Не сегодня.").show(evelynSad),
	)

	g.beat("s1.inside", bwHouseExterior,
		narr("В прихожей пахнет холодным деревом и сушёными яблоками. На вешалке осталось бабушкино пальто.").bg(bwKitchen).hide(),
		narr("Один рукав вывернут, будто его сняли торопливо."),
		narr("На кухне стоят круглый стол, два стула и старый радиоприёмник. Над мойкой — окно в сторону леса."),
		narr("Часы показывают половину четвёртого и не идут."),
		narr("Эвелин включает свет."),
		evelynSad.say("Я приехала."),
		narr("Ответа нет. Она опускает чемодан."),
		thought("Зачем я это сказала?"),
	)

	// Exploration hub: every item links back to s1.kitchen, which replays
	// the item's lines on arrival. Items are one-time (g.explore), so the
	// list empties as it's explored; the last entry is the way onward.
	explore := from("s1.inside", "s1.kitchen")
	g.beat("s1.kitchen", bwKitchen)
	g.explore(explore, "Пальто в прихожей", "s1.kitchen",
		narr("Эвелин поправляет вывернутый рукав. В кармане обнаруживается смятый список покупок: хлеб, батарейки, чай, яблоки.").hide(),
		narr("Последнее слово подчёркнуто дважды."),
		thought("Она собиралась покупать яблоки. У неё были планы на следующую неделю. Как это может просто закончиться?").show(evelynSad),
		narr("Эвелин возвращает список в карман."),
	)
	g.explore(explore, "Чашка возле мойки", "s1.cup")
	g.explore(explore, "Фотография на холодильнике", "s1.kitchen",
		narr("Маленькая Эвелин сидит у бабушки на плечах. За ними видны лес, деревянный настил и вода.").hide(),
		narr("На обороте написано: «Наше лето. Северный берег»."),
		thought("Я помню этот свитер. Помню, как она жаловалась, что я тяну её за волосы. Почему я совсем не помню озеро?").show(evelyn),
		narr("Эвелин кладёт фотографию в блокнот."),
	)
	g.explore(explore, "Конверт с документами", "s1.kitchen",
		narr("Внутри — бумаги на дом и копия медицинского заключения. Причина смерти: сердечный приступ.").hide(),
		narr("Подпись врача незнакома."),
		thought("Я прочитала это уже столько раз. Слова не меняются. Почему мне всё ещё кажется, что на следующем листе будет другое объяснение?").show(evelynSad),
		narr("Она возвращает заключение в конверт."),
	)
	g.link(explore, "Записка у телефона", "s1.note")

	g.beat("s1.cup", bwKitchen,
		narr("На дне засох тёмный ободок. Рядом лежит ложка.").hide(),
		narr("Чашка стоит на сложенном кухонном полотенце, словно бабушка собиралась вернуться и допить чай."),
		thought("Ты даже посуду не оставляла до утра. А я всё детство думала, что чашки моют себя сами.").show(evelynSad),
	)
	g.link(from("s1.cup"), "Вымыть чашку", "s1.kitchen",
		narr("Эвелин моет чашку, не задумываясь. А потом долго держит её в руках, не зная, куда поставить.").show(evelynSad),
	)
	g.link(from("s1.cup"), "Оставить как есть", "s1.kitchen",
		narr("Эвелин не трогает чашку. Ей кажется, что, убрав её, она окончательно признает: бабушка больше не вернётся.").show(evelynSad),
	)

	g.beat("s1.note", bwKitchen,
		narr("«Линни, если решишь продавать дом, сначала поговори с Морганом. Ключ с синей ниткой отдай ему лично.").hide(),
		narr("И перестань оставлять мокрые ботинки на дереве. Б.»"),
		narr("Под запиской висит небольшой латунный ключ. Он слишком узкий для входной двери."),
		narr("Эвелин смотрит на мокрые следы, которые оставила в коридоре."),
		evelynSad.say("Ну вот. Уже ругаешься."),
		thought("Ты писала так, будто просто уйдёшь в магазин. Почему нельзя было написать ещё одну строчку? Кто такой Морган? От чего ключ?"),
		narr("Она убирает ключ в карман куртки."),
		narr("Эвелин выключает свет на кухне.").show(evelyn),
		narr("В стекле на секунду возникает силуэт человека.").hide(),
		narr("Она резко оборачивается.").show(evelynWorried),
		narr("Никого."),
		narr("За окном, у границы леса, будто бы движется тёмная фигура."),
		thought("Ветка. Это могла быть ветка."),
	)
	g.link(from("s1.note"), "Подойти к окну", "s1.window",
		narr("Эвелин всматривается в лес. Только мокрые стволы и пустая дорожка.").hide(),
	)
	g.link(from("s1.note"), "Сразу задвинуть штору", "s1.window",
		narr("Эвелин задёргивает штору, не глядя в темноту."),
	)

	g.beat("s1.window", bwKitchen,
		narr("Она всё-таки проверяет, заперта ли входная дверь.").show(evelynWorried),
	)
}

// --- Сцена 2. Люди за дальним столом ---

func sceneTwo(g *storyGraph) {
	g.link(from("s1.window"), "Утром — в университет", "s2.cafeteria")

	g.beat("s2.cafeteria", bwCafeteria,
		narr("Университет занимает несколько старых кирпичных корпусов. Между ними лежит мокрый внутренний двор с клумбами и низкой каменной оградой.").hide(),
		narr("Столовая расположена в бывшем зимнем саду. Высокие окна запотели. Под потолком проходят трубы отопления, на подоконниках стоят большие растения с пыльными листьями."),
		narr("Здесь шумно: звенят подносы, скрипят стулья, кто-то спорит о пересдаче."),
		thought("Хорошо. Люди, которым страшнее всего завтрашний зачёт. Я могу посидеть среди них хотя бы полчаса.").show(evelyn),
		narr("Эвелин пытается открыть сэндвич. Упаковка рвётся, половина начинки падает на поднос."),
		noah.say("Поздравляю. Теперь он выглядит так же, как на вкус."),
		narr("Рядом останавливается парень с двумя стаканами кофе."),
		evelyn.say("Это официальная экскурсия для новеньких?"),
		noah.say("Нет. На официальной говорят, что еда нормальная."),
		narr("Он ставит один стакан перед ней."),
		noah.say("Ной Картер. Кофе лишний."),
		evelyn.say("Ты всегда покупаешь кофе незнакомым людям?"),
		noah.say("Только тем, кто уже пострадал от нашего сэндвича."),
	).titled("Люди за дальним столом")

	g.link(from("s2.cafeteria"), "Принять помощь", "s2.hall",
		evelyn.say("Эвелин Морроу. И, кажется, ты только что спас мне завтрак."),
		noah.say("Не преувеличивай. Кофе тоже отсюда."),
		narr("Он садится без прежней неловкости."),
		thought("С ним легко. Я уже успела забыть, как это бывает.").show(evelyn),
		narr("Ной охотно рассказывает о городе, а потом сам предлагает обменяться номерами.").show(noah),
	)
	g.link(from("s2.cafeteria"), "Держать дистанцию", "s2.hall",
		evelyn.say("Спасибо, но я предпочитаю сама выбирать, чем отравиться."),
		noah.say("Уважаю."),
		narr("Он оставляет стакан на краю стола."),
		noah.say("Если передумаешь, он пока ничей."),
		thought("Хорошо, что он не обиделся. Я не хочу быть грубой. Просто сил на новых людей почти нет.").show(evelynSad),
	)
	g.link(from("s2.cafeteria"), "Сразу спросить о Моргане", "s2.hall",
		evelyn.say("Ты знаешь профессора Моргана?"),
		noah.say("Историка? Да. Если он не на лекции, ищи среди коробок в архиве."),
		evelynSad.say("Мне нужно отдать ему кое-что от бабушки."),
		narr("Ной замечает, как она произносит последнее слово, и меняет тон.").show(noah),
		noah.say("Я узнаю, когда он будет. Чтобы ты зря не ходила."),
	)

	look := from("s2.hall")
	g.beat("s2.hall", bwCafeteria).talksWith(bwNoahID, noah.sprite, bwTalkMessages)
	g.explore(look, "Доска объявлений", "s2.board")
	g.explore(look, "Дальний стол", "s2.hall",
		narr("За дальним столом сидят трое: Люси, Адриан и Дэмиан Вейлы.").hide(),
		narr("У Люси раскрыта тетрадь.").show(lucy),
		narr("Дэмиан лениво вращает кольцо на пальце.").show(damian),
		narr("Адриан сидит вполоборота к залу.").show(adrian),
		narr("Еда перед ними почти не тронута."),
		thought("Я тоже ничего не съела. Это ещё не делает людей загадочными.").show(evelyn),
		narr("Но Эвелин замечает: когда у соседнего стола с грохотом падает поднос, все трое поворачиваются одновременно.").hide(),
	)
	g.explore(look, "Присмотреться к Люси", "s2.lucy")
	g.explore(look, "Присмотреться к Дэмиану", "s2.damian")
	g.link(look, "Спросить Ноя о людях за дальним столом", "s2.vails")

	// An optional side beat so the player can talk to Damian, who otherwise only
	// shows up for one line in s2.meet.
	g.beat("s2.damian", bwCafeteria,
		narr("Дэмиан замечает её взгляд раньше, чем она успевает отвернуться.").show(damian),
		damian.say("Если хочешь что-то спросить — спрашивай. Я не кусаюсь. Пока не доказано обратное."),
		narr("Кольцо на его пальце делает ещё один оборот."),
	).talksWith(bwDamianID, damian.sprite, bwTalkMessages)
	g.link(from("s2.damian"), "Вернуться к Ною", "s2.hall",
		evelyn.say("Не сегодня."),
		damian.say("Как знаешь."),
	)

	// Like s2.damian: the player may talk to Lucy before the scheduled meeting.
	g.beat("s2.lucy", bwCafeteria,
		narr("Стоит Эвелин задержать на ней взгляд, Люси поднимает голову и улыбается.").show(lucy),
		thought("Она заметила меня раньше, чем я решила на неё посмотреть. Или мне просто неловко, что я пялюсь."),
	).talksWith(bwLucyID, lucy.sprite, bwTalkMessages)
	g.link(from("s2.lucy"), "Вернуться к Ною", "s2.hall",
		narr("Люси машет ей и возвращается к тетради.").show(lucy),
	)

	g.beat("s2.board", bwCafeteria,
		narr("Среди объявлений о секциях и продаже учебников висит заметка об ограничении доступа к северной лесной дороге после дождей.").hide(),
		thought("Северный берег на фотографии. Северная дорога здесь. Может быть, это вообще разные места. Надо спросить.").show(evelyn),
	)
	g.link(from("s2.board"), "Спросить Ноя о дороге", "s2.hall",
		evelyn.say("Куда ведёт северная дорога?"),
		noah.say("К озеру. После дождей её закрывают — размывает."),
	)
	g.link(from("s2.board"), "Вернуться за стол", "s2.hall")

	g.beat("s2.vails", bwCafeteria,
		evelyn.say("А это кто?"),
		noah.say("Вейлы."),
		evelyn.say("По твоему тону я должна была всё понять."),
		noah.say("Просто держись с ними осторожнее."),
	)
	g.link(from("s2.vails"), "Попросить факты", "s2.approach",
		evelyn.say("Они кому-нибудь что-то сделали?"),
		noah.say("Мне — нет."),
		evelyn.say("Тогда почему?"),
		narr("Ной думает.").show(noah),
		noah.say("Ладно. У меня нет хорошего ответа. Они мне не нравятся. Это ещё не значит, что обязаны не нравиться тебе."),
		thought("Хотя бы он отличает одно от другого.").show(evelyn),
	)
	g.link(from("s2.vails"), "Согласиться, что они странные", "s2.approach",
		evelyn.say("Они действительно ведут себя странно."),
		noah.say("Подожди, пока поговоришь с Адрианом."),
		narr("В этот момент Адриан поворачивается в их сторону.").show(adrian),
		narr("Между столами слишком далеко, чтобы услышать негромкий разговор."),
		thought("Он услышал своё имя? Или просто посмотрел сюда?"),
	)
	g.link(from("s2.vails"), "Не обсуждать незнакомых людей", "s2.approach",
		evelyn.say("Я сначала сама познакомлюсь."),
		noah.say("Справедливо."),
		narr("Он отодвигается, освобождая проход."),
		noah.say("Если что, я буду здесь. Продолжать героически завтракать."),
	)

	g.beat("s2.approach", bwCafeteria)
	g.link(from("s2.approach"), "Подойти к Вейлам", "s2.meet",
		narr("Эвелин подходит к дальнему столу. Первой её приветствует Люси.").show(lucy),
	)
	g.link(from("s2.approach"), "Остаться на месте", "s2.meet",
		narr("Эвелин остаётся на месте. Чуть позже Люси сама подходит к её столу.").show(lucy),
	)

	g.beat("s2.meet", bwCafeteria,
		lucy.say("Эвелин, да? Морган говорил, что ты приедешь."),
		evelyn.say("Кажется, я единственный человек в городе, который не знает, кто такой Морган."),
		narr("Люси смеётся.").show(lucy),
		narr("Рядом появляется Адриан.").show(adrian),
		adrian.say("Люси, нам пора."),
		lucy.say("У нас ещё десять минут."),
		narr("Эвелин достаёт фотографию из блокнота.").show(evelyn),
		evelyn.say("Тогда у меня короткий вопрос. Где это?"),
		narr("Люси наклоняется.").show(lucy),
		lucy.say("Озеро Без Эха. Северный берег."),
		narr("Адриан смотрит на бабушку на снимке.").show(adrian),
		narr("Его выражение меняется так быстро, что Эвелин не успевает понять как."),
		adrian.say("Там давно нечего делать."),
		evelyn.say("Часто бываешь?"),
		adrian.say("Я никогда не был у озера."),
		narr("Дэмиан, проходящий мимо с подносом, задерживается.").show(damian),
		damian.say("Тогда откуда тебе знать?"),
		narr("Адриан не отвечает.").show(adrian),
		thought("Кажется, я случайно попала в чужую ссору.").show(evelyn),
		narr("Вейлы уходят. Люси оборачивается и прощально поднимает руку.").show(lucy),
	)
}

// --- Сцена 3. Расстояние в пятьдесят метров ---

// sceneThree returns the conditions "the hood was photographed in this
// playthrough" and its negation, used again in scene 4. The chapter's entry
// choice is part of the condition so that a photo taken in an earlier
// playthrough doesn't count (see lineCondition).
func sceneThree(g *storyGraph) (hasPhoto, noPhoto lineCondition) {
	entry := g.link(from("s2.meet"), "Вечером — на автобусную остановку", "s3.parking")[0]

	g.beat("s3.parking", bwParking,
		narr("К вечеру дождь становится сильнее. По асфальту текут тонкие ручьи, отражающие красный свет автомобильных фар.").hide(),
		narr("От главного входа до автобусной остановки нужно пересечь парковку. Слева — спуск с боковой улицы, справа — другой университетский корпус."),
		narr("Под его навесом стоит Адриан.").show(adrian),
		narr("Эвелин замечает его и отворачивается.").show(evelyn),
		thought("Не собираюсь проверять, смотрит ли он на меня. Я здесь второй день. Можно хотя бы неделю прожить без странных увлечений."),
		narr("Телефон вибрирует. Сообщение от Ноя:"),
		narr("«Морган завтра до обеда в архиве. Нашёл для тебя»."),
		narr("Эвелин убирает телефон."),
		narr("Скрежет тормозов.").hide(),
		narr("Автомобиль срывается на мокром повороте."),
		narr("Она успевает увидеть лицо водителя и собственное отражение в лобовом стекле."),
		narr("Кто-то хватает её за запястье."),
		narr("Удар."),
		narr("Адриан стоит между ней и машиной. Его ладонь упирается в капот. Металл продавлен вокруг пальцев.").show(adrian),
		thought("Он был у корпуса."),
		narr("Следом приходит другое изображение."),
		narr("Тёмная комната. Синий свет. Бабушка стоит возле деревянного стола. На рукаве кровь.").bg(bwForesterInterior).hide(),
		grandma.say("Не отдавай…"),
		narr("Всё исчезает."),
		narr("Адриан резко отпускает Эвелин.").bg(bwParking).show(adrian),
		adrian.say("Ты ударилась?"),
	).titled("Расстояние в пятьдесят метров")

	// Whichever reaction comes first, Evelyn brings up the vision before
	// Adrian leaves.
	mentionLater := []scriptLine{
		narr("Адриан уже собирается уйти.").show(adrian),
		evelynWorried.say("Подожди. Я видела свою бабушку."),
		narr("Его лицо меняется.").show(adrian),
		adrian.say("Тебе нужен врач."),
	}
	g.link(from("s3.parking"), "Как ты успел?", "s3.aftermath", append([]scriptLine{
		evelynWorried.say("Как ты успел?"),
		adrian.say("Я был рядом."),
		evelynWorried.say("Нет. Я тебя видела."),
		narr("Он смотрит через парковку.").show(adrian),
		adrian.say("Сейчас это неважно."),
		thought("Для него неважно. А у меня машина стоит там, где секунду назад должна была быть я.").show(evelynWorried),
	}, mentionLater...)...)
	g.link(from("s3.parking"), "Попросить его не уходить", "s3.aftermath", append([]scriptLine{
		evelynWorried.say("Подожди. Я… не могу нормально вдохнуть."),
		narr("Адриан отступает от машины и остаётся рядом.").show(adrian),
		adrian.say("Посмотри на меня. Медленнее. Ты уже в безопасности."),
		narr("Он говорит спокойно, но держит руки вдоль тела."),
		thought("Почему он стоит так, будто боится задеть меня?"),
	}, mentionLater...)...)
	g.link(from("s3.parking"), "Рассказать о видении", "s3.aftermath",
		evelynWorried.say("Я видела свою бабушку."),
		narr("Его лицо меняется.").show(adrian),
		adrian.say("Что?"),
		evelynWorried.say("Только что."),
		narr("Он медленно отпускает край её мокрого рукава.").show(adrian),
		adrian.say("Тебе нужен врач."),
	)

	g.beat("s3.aftermath", bwParking,
		narr("Ной прибегает от ступеней. Водитель выбирается из машины.").hide(),
	)

	explore := from("s3.aftermath", "s3.lot")
	g.beat("s3.lot", bwParking)
	g.explore(explore, "Капот", "s3.hood")
	g.explore(explore, "Место под навесом", "s3.lot",
		narr("С парковки видно, где стоял Адриан. Между ним и местом аварии несколько рядов машин.").hide(),
		thought("Он не мог просто шагнуть ко мне. Я не настолько плохо оцениваю расстояние.").show(evelynWorried),
	)
	g.explore(explore, "Поговорить с Ноем", "s3.noah")
	g.explore(explore, "Поговорить с водителем", "s3.driver")
	g.link(explore, "Оглянуться на Адриана", "s3.gone")

	g.beat("s3.hood", bwParking,
		narr("В металле различимы следы ладони.").hide(),
		thought("Это можно проверить. Не ощущение, не вспышка перед глазами. Настоящий металл.").show(evelyn),
	)
	photoTaken := g.link(from("s3.hood"), "Сфотографировать вмятину", "s3.lot",
		narr("Эвелин фотографирует смятый капот."),
	)
	photoSkipped := g.link(from("s3.hood"), "Не фотографировать", "s3.lot",
		narr("Эвелин убирает телефон, так и не сделав снимок."),
	)
	hasPhoto = chosen(photoTaken, append(photoSkipped, entry)...)
	noPhoto = notChosen(photoTaken, append(photoSkipped, entry)...)

	g.beat("s3.noah", bwParking,
		evelyn.say("Ты видел, как он оказался рядом?"),
		noah.say("Нет. Я услышал удар и выбежал."),
	).talksWith(bwNoahID, noah.sprite, bwTalkMessages)
	g.link(from("s3.noah"), "Настаивать", "s3.lot",
		noah.say("Я верю, что ты что-то увидела. Но не буду говорить, что видел это тоже."),
		thought("Неприятно. Но лучше, чем если бы он просто согласился.").show(evelyn),
	)
	g.link(from("s3.noah"), "Не настаивать", "s3.lot")

	g.beat("s3.driver", bwParking,
		narr("Водитель стоит у смятого капота.").show(carDriver),
	)
	g.link(from("s3.driver"), "Обвинить его", "s3.lot",
		evelynWorried.say("Вы вообще смотрели на дорогу?"),
		carDriver.say("Я тормозил! Меня понесло. Я не…"),
		narr("Он замолкает, увидев, что у неё дрожат руки."),
	)
	g.link(from("s3.driver"), "Спросить спокойно", "s3.lot",
		evelyn.say("Вы заметили человека перед машиной?"),
		carDriver.say("Только в последний момент. Я не понял, откуда он взялся."),
		thought("Ещё один человек видел его. Но никто не видел достаточно.").show(evelyn),
	)

	g.beat("s3.gone", bwParking,
		narr("Адриан уходит, пока Ной разговаривает с водителем.").hide(),
		narr("Эвелин замечает это слишком поздно.").show(evelynWorried),
		narr("На экране телефона остаётся фотография смятого капота.").when(hasPhoto),
	)
	return hasPhoto, noPhoto
}

// --- Сцена 4. Версия, в которую удобно поверить ---

func sceneFour(g *storyGraph, hasPhoto, noPhoto lineCondition) {
	g.link(from("s3.gone"), "Утром — в старый корпус", "s4.corridor")

	g.beat("s4.corridor", bwCorridor,
		narr("Утром здесь почти пусто. В высоких окнах серое небо. Батареи потрескивают, на стенах висят фотографии выпускников.").hide(),
		narr("Адриан стоит у подоконника с раскрытой книгой.").show(adrian),
		narr("Эвелин замечает его раньше, чем он поднимает голову."),
	).titled("Версия, в которую удобно поверить")

	watch := from("s4.corridor", "s4.watch")
	g.beat("s4.watch", bwCorridor).talksWith(bwAdrianID, adrian.sprite, bwTalkMessages)
	g.explore(watch, "Присмотреться к книге", "s4.watch",
		narr("За то время, пока Эвелин идёт по коридору, он ни разу не переворачивает страницу.").show(adrian),
		thought("Может, он тоже плохо спал. Почему мне вообще хочется найти что-то, что нас объединяет?"),
	)
	g.explore(watch, "Присмотреться к его рукам", "s4.watch",
		narr("На пальцах нет повязок, ссадин или синяков.").show(adrian),
		thought("Он остановил машину. Даже если я неправильно поняла всё остальное — руке должно быть больно."),
	)
	g.link(watch, "Заговорить с ним", "s4.talk")

	g.beat("s4.talk", bwCorridor,
		evelyn.say("Врач сказал, с головой у меня всё в порядке."),
		narr("Адриан закрывает книгу.").show(adrian),
		adrian.say("Хорошо."),
		evelyn.say("Это портит твою версию?"),
		adrian.say("Ты испугалась. В такие моменты память…"),
		evelyn.say("Осторожнее со следующим словом."),
	)
	g.link(from("s4.talk"), "Предъявить свидетельство", "s4.key",
		evelyn.say("Ты оставил руку в металле.").when(hasPhoto),
		narr("Эвелин показывает ему снимок.").when(hasPhoto),
		adrian.say("Я вижу повреждённый капот.").when(hasPhoto),
		evelyn.say("У тебя даже царапины нет.").when(hasPhoto),
		narr("Он убирает руки в карманы.").show(adrian).when(hasPhoto),
		evelyn.say("Я видела вмятину. Она повторяла твою ладонь.").when(noPhoto),
		adrian.say("Ты стояла рядом с машиной после удара. Неудивительно, что всё смешалось.").when(noPhoto),
		thought("Он использует именно то, в чём я сама сомневаюсь.").show(evelynWorried).when(noPhoto),
		narr("С этого момента Адриан осторожнее и отвечает короче.").show(adrian),
	)
	g.link(from("s4.talk"), "Начать с благодарности", "s4.key",
		evelyn.say("Ты спас меня. Я ещё не сказала спасибо."),
		narr("Он явно ожидал другого.").show(adrian),
		adrian.say("Не за что."),
		evelyn.say("Есть за что. Но я всё равно хочу понять, что произошло."),
		adrian.say("Я понимаю."),
		narr("Он остаётся в разговоре и перестаёт притворяться, что торопится."),
	)
	g.link(from("s4.talk"), "Спросить о бабушке", "s4.key",
		evelyn.say("Почему ты испугался, когда я сказала про бабушку?"),
		adrian.say("Я не испугался."),
		evelyn.say("Тогда посмотри на меня и скажи, что не знал её."),
		narr("Он смотрит. Но нужных слов не произносит.").show(adrian),
		thought("Вот. Я не получила ответа. Но он хотя бы перестал отрицать всё подряд.").show(evelyn),
	)

	g.beat("s4.key", bwCorridor,
		narr("Эвелин достаёт ключ с синей ниткой.").show(evelyn),
		evelyn.say("Она оставила мне это."),
		narr("Адриан сразу узнаёт его.").show(adrian),
		adrian.say("Отдай Моргану."),
		evelyn.say("Я не говорила, что собираюсь к нему."),
		narr("Пауза."),
	)
	g.link(from("s4.key"), "Надавить", "s4.leave",
		evelyn.say("Сколько ещё ты знаешь обо мне?"),
		adrian.say("Меньше, чем ты думаешь."),
		evelyn.say("Очень удобная формулировка."),
	)
	g.link(from("s4.key"), "Дать ему объясниться", "s4.leave",
		evelynSad.say("Если ты знал её, просто скажи. Мне сейчас не так много людей могут о ней рассказать."),
		narr("Он отворачивается к окну.").show(adrian),
		adrian.say("Я знал её."),
		narr("Это первое прямое признание."),
	)

	g.beat("s4.leave", bwCorridor,
		evelyn.say("Чего ты от меня хочешь?"),
		adrian.say("Чтобы ты уехала."),
		thought("Вот это прозвучало честно. Жаль, что именно это.").show(evelynSad),
		narr("Дверь аудитории открывается. Коридор наполняется студентами.").hide(),
		narr("Адриан уходит вместе с потоком."),
	)
}

// --- Сцена 5. Чужая подпись ---

func sceneFive(g *storyGraph) {
	g.link(from("s4.leave"), "Спуститься в архив", "s5.archive")

	g.beat("s5.archive", bwArchive,
		narr("Архив находится в полуподвале. Узкие окна расположены почти под потолком, и с улицы через них видны только ноги прохожих.").hide(),
		narr("В комнате пахнет бумагой, пылью и кофе. Вдоль стен стоят высокие металлические стеллажи. На столе у окна — старая настольная лампа с зелёным абажуром."),
		narr("Профессор Морган сидит среди коробок. Когда входит Эвелин, он встаёт слишком быстро и задевает коленом стол.").show(morgan),
		morgan.say("Простите. Вы очень на неё похожи."),
		evelyn.say("Все говорят."),
		morgan.say("Она бы рассердилась. Считала, что люди обязаны быть похожи на себя."),
		narr("Эвелин улыбается.").show(evelyn),
		thought("Он действительно её знал."),
		narr("Она кладёт ключ на стол."),
		narr("Улыбка Моргана исчезает.").show(morgan),
		morgan.say("Она оставила его вам?"),
		evelyn.say("Просила отдать лично."),
		narr("Он протягивает руку.").show(morgan),
	).titled("Чужая подпись")

	keyGiven := g.link(from("s5.archive"), "Отдать ключ", "s5.room",
		narr("Морган берёт ключ осторожно, проводит большим пальцем по синей нитке.").show(morgan),
		evelyn.say("Что он открывает?"),
		morgan.say("Старый дом лесника. На северном берегу озера."),
		narr("Он держит его так, будто это не ключ, а что-то, что можно сломать.").show(morgan),
	)
	keyKept := g.link(from("s5.archive"), "Придержать ключ", "s5.room",
		narr("Эвелин накрывает ключ ладонью.").show(evelyn),
		evelyn.say("Сначала объясните."),
		narr("Морган убирает руку.").show(morgan),
		morgan.say("Справедливо."),
		narr("Он рассказывает о старом доме лесника на северном берегу озера, но следит за тем, куда Эвелин смотрит."),
	)

	explore := from("s5.room")
	g.beat("s5.room", bwArchive).talksWith(bwMorganID, morgan.sprite, bwTalkMessages)
	g.explore(explore, "Карта окрестностей", "s5.room",
		narr("На стене висит карта с пометками карандашом. Северный берег озера обведён.").hide(),
		narr("К нему ведёт лесная дорога, заканчивающаяся возле маленького прямоугольника."),
		thought("Дом лесника. Теперь у этого ключа есть адрес.").show(evelyn),
		narr("Эвелин фотографирует карту."),
	)
	g.explore(explore, "Коробка «Морроу»", "s5.box")
	g.link(explore, "Журнал посещений дома лесника", "s5.journal")

	g.beat("s5.box", bwArchive,
		narr("Внутри — газетные вырезки, письма, старые планы земельных участков.").hide(),
		narr("На верхнем листе бабушкиным почерком написано: «Морган, это третья копия. Не потеряй хотя бы её»."),
		thought("Она и с ним так разговаривала.").show(evelyn),
	)
	g.link(from("s5.box"), "Показать лист Моргану", "s5.room",
		narr("Профессор усмехается.").show(morgan),
		morgan.say("Первые две я, кстати, нашёл."),
		narr("На мгновение разговор становится обычным и человеческим."),
	)
	g.link(from("s5.box"), "Положить лист обратно", "s5.room")

	g.beat("s5.journal", bwArchive,
		narr("Последняя заполненная страница датирована днём смерти бабушки.").hide(),
		narr("Время: 23:10."),
		narr("Подпись бабушки. Ниже: «А. Вейл»."),
		thought("Инициал — не имя. Я пока не знаю, что это Адриан. Но сколько Вейлов нужно встретить за два дня, чтобы перестать считать это совпадением?").show(evelynWorried),
		narr("Эвелин поворачивает журнал к Моргану."),
		evelyn.say("Она умерла дома."),
		narr("Морган молчит.").show(morgan),
		evelyn.say("Так мне сказали."),
		morgan.say("Её нашли дома."),
		evelyn.say("Это разные вещи."),
		narr("Он снимает очки.").show(morgan),
	)
	g.link(from("s5.journal"), "Попросить по-человечески", "s5.goodbye",
		evelynSad.say("Я даже не успела с ней попрощаться. Пожалуйста, не заставляйте меня выпрашивать каждое слово."),
		narr("Морган долго смотрит на журнал.").show(morgan),
		morgan.say("Она работала там в тот вечер. Я должен был приехать, но не смог."),
		evelyn.say("Почему?"),
		morgan.say("Решил, что успею завтра."),
		narr("Он уже много раз отвечал на этот вопрос самому себе."),
	)
	g.link(from("s5.journal"), "Обвинить во лжи", "s5.goodbye",
		evelynWorried.say("Вы все договорились? Врач, Вейлы, вы?"),
		morgan.say("Я не знаю, что вам говорил врач."),
		evelynWorried.say("А что не договорили вы, знаете?"),
		narr("Он закрывает журнал.").show(morgan),
		morgan.say("Да."),
		narr("Ответ лишает Эвелин следующей реплики."),
		narr("Больше Морган о личном не говорит."),
	)
	g.link(from("s5.journal"), "Держаться фактов", "s5.goodbye",
		evelyn.say("Кто такой А. Вейл?"),
		morgan.say("Адриан."),
		evelyn.say("Вы видели его там в тот вечер?"),
		morgan.say("Нет."),
		evelyn.say("Тогда вы знаете о его присутствии только по журналу?"),
		morgan.say("По журналу — и с его слов."),
		thought("Теперь хотя бы понятно, откуда что известно.").show(evelyn),
	)

	g.beat("s5.goodbye", bwArchive,
		narr("Морган возвращает ключ.").show(morgan).when(chosen(keyGiven, keyKept...)),
		morgan.say("Там остались её записи. Я отвезу вас в субботу."),
		evelyn.say("Через четыре дня?"),
		morgan.say("Мне нужно время."),
		evelyn.say("Мне тоже. Но вы уже решили, что можете его у меня забрать."),
		narr("Она забирает ключ.").show(evelyn),
		narr("У двери профессор окликает её.").show(morgan),
		morgan.say("Не ездите туда одна."),
	)
	g.link(from("s5.goodbye"), "«Тогда поедем сегодня»", "s5.door",
		evelyn.say("Тогда поедем сегодня."),
		morgan.say("Сегодня я не смогу."),
	)
	g.link(from("s5.goodbye"), "«Я ничего не обещаю»", "s5.door",
		evelyn.say("Я ничего не обещаю."),
		narr("По его лицу видно: он понимает, что она, скорее всего, поедет.").show(morgan),
	)
	g.link(from("s5.goodbye"), "Ничего не сказать", "s5.door",
		narr("Эвелин молча уходит, оставив его смотреть на открытый журнал.").hide(),
	)
	g.beat("s5.door", bwArchive)
}

// --- Сцена 6. Третья ступенька ---

func sceneSix(g *storyGraph) {
	g.link(from("s5.door"), "Вечером — на крыльцо", "s6.porch")

	g.beat("s6.porch", bwHouseExterior,
		narr("К вечеру свет на улице становится зеленовато-серым. Дождь прекратился, но с крыши ещё капает.").hide(),
		narr("Эвелин проверяет телефон, карту и фонарик. Ключ с синей ниткой лежит во внутреннем кармане.").show(evelyn),
		thought("Я собираюсь ехать в заброшенный дом после предупреждения двух людей. Если бы это делала моя подруга, я бы назвала её идиоткой."),
		narr("Она застёгивает куртку."),
		thought("Но это не её бабушка."),
		narr("Скрипит калитка."),
		narr("На дорожке стоит Адриан.").show(adrian),
		evelyn.say("Морган позвонил?"),
		adrian.say("Да."),
		evelyn.say("В следующий раз пусть попробует нормально ответить на вопросы. Это экономит время всем."),
		adrian.say("Дорога размыта."),
		evelyn.say("Это должно меня остановить?"),
		adrian.say("Нет. Поэтому я здесь."),
	).titled("Третья ступенька")

	g.link(from("s6.porch"), "Поехать вместе", "s6.road",
		evelyn.say("Хорошо. Но по дороге ты расскажешь, что делал с ней у озера."),
		adrian.say("Постараюсь."),
		evelyn.say("Мне уже не нравится это слово."),
		thought("По крайней мере, теперь он будет в машине и не сможет просто уйти посреди разговора."),
	)
	g.link(from("s6.porch"), "Поехать, но написать Ною", "s6.road",
		evelyn.say("Я отправлю Ною, куда еду и с кем."),
		narr("Адриан кивает.").show(adrian),
		adrian.say("Отправь."),
		evelyn.say("Ты даже не возражаешь?"),
		adrian.say("Это разумно."),
		narr("Ответ приходит почти сразу: «Жду сообщения, когда вернёшься»."),
		thought("Хорошо, что хотя бы один человек знает, где меня искать.").show(evelyn),
	)
	g.link(from("s6.porch"), "Отказаться от его машины", "s6.lodge",
		evelyn.say("Я доберусь сама."),
		adrian.say("На третьем повороте не сворачивай направо. На старых картах там дорога. Сейчас — ручей."),
		narr("Эвелин добирается на такси до лесного съезда. Дальше идёт пешком.").bg(bwForestRoad).hide(),
		narr("Тропа ведёт к одноэтажному дому на невысоком склоне. Между стволами поблёскивает вода."),
		narr("Адриан встречает её у дома лесника.").bg(bwForesterExterior).show(adrian),
		evelyn.say("Я отказалась от сопровождения."),
		adrian.say("Я тоже должен проверить дом."),
	)

	g.beat("s6.road", bwHouseExterior,
		narr("Большую часть пути между ними слышны только дворники.").bg(bwCarInterior).hide(),
		narr("Лес подступает к дороге вплотную. В низинах лежит туман. Иногда фары выхватывают старые указатели, надписи на которых почти стёрлись."),
		narr("Адриан сбрасывает скорость перед поворотами ещё до того, как они появляются в свете фар.").show(adrian),
		thought("Он хорошо знает дорогу. Хотя у озера якобы никогда не был.").show(evelyn),
	).talksWith(bwAdrianID, adrian.sprite, bwTalkMessages)
	arrival := []scriptLine{
		narr("Дальше дорогу перекрывает упавшая сосна.").bg(bwForestRoad).hide(),
		narr("За ней начинается тропа к одноэтажному дому на невысоком склоне. Между стволами поблёскивает вода."),
	}
	g.link(from("s6.road"), "Спросить о бабушке", "s6.lodge", append([]scriptLine{
		evelyn.say("Она тебе нравилась?"),
		adrian.say("Она никому не позволяла решать за неё."),
		evelyn.say("Это не ответ."),
		narr("Он ненадолго смотрит на неё.").show(adrian),
		adrian.say("Да. Нравилась."),
		evelyn.say("Она и тебя ругала за мокрые ботинки?"),
		narr("На его лице появляется короткая улыбка.").show(adrian),
		adrian.say("За грязь на ступеньках."),
		thought("Я не хочу, чтобы он так о ней говорил. И одновременно хочу, чтобы не замолкал.").show(evelynSad),
	}, arrival...)...)
	g.link(from("s6.road"), "Спросить об озере", "s6.lodge", append([]scriptLine{
		evelyn.say("Почему «Без Эха»?"),
		adrian.say("По легенде оно ничего не возвращает."),
		evelyn.say("Даже эхо?"),
		adrian.say("Особенно то, что люди хотят услышать снова."),
		narr("Эвелин смотрит в окно.").show(evelyn),
		thought("Он пересказывает легенду или предупреждает меня?"),
	}, arrival...)...)
	g.link(from("s6.road"), "Промолчать", "s6.lodge", append([]scriptLine{
		narr("Эвелин отворачивается. Адриан не пытается заполнить тишину.").hide(),
		thought("Спасибо хотя бы за это. Мне нужно несколько минут, чтобы не смотреть на него и не думать о подписи в журнале.").show(evelyn),
	}, arrival...)...)

	g.beat("s6.lodge", bwForesterExterior,
		narr("Одно окно заколочено. Крыша над крыльцом просела. Возле стены лежит ржавая лодочная цепь.").bg(bwForesterExterior).hide(),
		narr("В отличие от бабушкиного дома, этот выглядит так, будто его давно перестали ждать."),
	)

	explore := from("s6.lodge", "s6.yard")
	g.beat("s6.yard", bwForesterExterior)
	g.explore(explore, "Следы у крыльца", "s6.yard",
		narr("В грязи видны наложившиеся друг на друга следы обуви.").hide(),
		thought("Слишком размыты, чтобы понять, когда здесь ходили. Но место явно не забыто всеми.").show(evelyn),
	)
	g.explore(explore, "Озеро между деревьями", "s6.yard",
		narr("Поверхность почти неподвижна. Ветки качаются, а вода остаётся гладкой.").hide(),
		thought("Берег может быть защищён от ветра. Наверняка есть нормальное объяснение.").show(evelynWorried),
	)
	g.explore(explore, "Дверь", "s6.yard",
		narr("Вокруг замочной скважины свежие царапины поверх старой краски.").hide(),
		thought("Кто-то недавно промахивался ключом. В темноте? В спешке?").show(evelynWorried),
	)
	g.link(explore, "Подняться на крыльцо", "s6.steps")

	g.beat("s6.steps", bwForesterExterior,
		narr("Эвелин ступает на крыльцо.").show(evelyn),
		adrian.say("Не на третью ступеньку. Доска…"),
		narr("Он замолкает.").show(adrian),
	)
	g.link(from("s6.steps"), "Напомнить о разговоре в столовой", "s6.threshold",
		evelyn.say("Ты никогда не был у озера."),
		narr("Адриан смотрит на неё.").show(adrian),
		evelyn.say("Это ты сказал. Вчера."),
		adrian.say("Я соврал."),
		evelyn.say("Зачем?"),
		adrian.say("Надеялся, что на этом разговор закончится."),
		evelyn.say("Как видишь, стратегия не сработала."),
		thought("Он признался сразу. От этого почему-то не становится легче."),
	)
	g.link(from("s6.steps"), "Спросить без обвинений", "s6.threshold",
		evelyn.say("Ты хорошо знаешь этот дом?"),
		adrian.say("Достаточно."),
		evelyn.say("Тогда внутри объяснишь, откуда."),
		narr("Он кивает.").show(adrian),
	)
	g.link(from("s6.steps"), "Не обратить внимания", "s6.threshold",
		narr("Эвелин переступает через третью ступеньку и открывает дверь.").hide(),
	)
	g.beat("s6.threshold", bwForesterExterior)
}

// --- Сцена 7. То, что осталось после неё ---

func sceneSeven(g *storyGraph) {
	g.link(from("s6.threshold"), "Войти в дом", "s7.room")

	g.beat("s7.room", bwForesterInterior,
		narr("Внутри одна большая комната и узкий проход в кладовую.").hide(),
		narr("Слева стоит железная печь. Напротив двери — деревянный стол, два стула и настенная металлическая оправа, похожая на крепление для небольшого зеркала."),
		narr("Справа тянется шкаф с перекошенными дверцами. Окно над столом выходит к озеру."),
		narr("Пахнет мокрой древесиной, холодной золой и чем-то металлическим."),
		narr("Эвелин направляет фонарик на стол и замирает.").show(evelynWorried),
		thought("Я видела его."),
		narr("Она делает шаг в сторону. Меняется угол, но узнавание не проходит."),
		thought("На парковке. Этот стол был там. Во вспышке."),
		adrian.say("Что?"),
		evelynWorried.say("Пока не знаю."),
		narr("Он остаётся возле двери, позволяя ей осмотреться.").show(adrian),
	).titled("То, что осталось после неё")

	explore := from("s7.room", "s7.search")
	g.beat("s7.search", bwForesterInterior)
	g.explore(explore, "Керосиновая лампа", "s7.search",
		narr("Лампа стоит на столе. На стекле сохранилась копоть.").hide(),
		narr("Эвелин приподнимает её. Под основанием дерево светлее, чем вокруг."),
		thought("Она стояла здесь долго. В видении тоже была лампа. Слева от бабушки.").show(evelynWorried),
	)
	g.explore(explore, "Книга на подоконнике", "s7.book")
	g.explore(explore, "Пятно на полу", "s7.stain")
	g.explore(explore, "Пустая оправа", "s7.search",
		narr("Она прикручена к стене четырьмя потемневшими болтами. Внизу застрял осколок синего стекла.").hide(),
		narr("Когда луч фонарика попадает на него, по комнате пробегает холодный свет."),
		narr("Эвелин резко выдыхает.").show(evelynWorried),
		thought("Вот этот цвет. Именно этот."),
	)
	g.explore(explore, "Кладовая", "s7.pantry")
	g.link(explore, "Вернуться к столу", "s7.vision")

	g.beat("s7.book", bwForesterInterior,
		narr("Между страницами засушен клевер. На полях бабушкиным почерком написано: «Слишком красиво, чтобы быть точным».").hide(),
		narr("Эвелин проводит пальцем рядом с надписью.").show(evelynSad),
		thought("Она спорила даже с книгами."),
	)
	g.link(from("s7.book"), "Забрать книгу", "s7.search",
		adrian.say("Она твоя."),
		evelyn.say("Я не спрашивала разрешения."),
		narr("Он опускает взгляд.").show(adrian),
	)
	g.link(from("s7.book"), "Оставить на месте", "s7.search",
		narr("Эвелин возвращает книгу на подоконник."),
	)

	g.beat("s7.stain", bwForesterInterior,
		narr("Под столом одна доска темнее остальных. Дерево вокруг высветлено, будто его долго тёрли.").hide(),
		thought("Может быть, пролитое масло. Краска. Я не могу посмотреть на пятно и решить, что это кровь.").show(evelynWorried),
		narr("Но ей вспоминается бабушкин рукав."),
	)
	g.link(from("s7.stain"), "Спросить Адриана", "s7.search",
		evelyn.say("Что здесь произошло?"),
		adrian.say("Ты уже что-то вспомнила?"),
		evelyn.say("Я никогда здесь не была. По крайней мере, не помню этого."),
		narr("Он молчит.").show(adrian),
		thought("Он не удивился вопросу. Его интересует, сколько я уже знаю.").show(evelynWorried),
	)
	g.link(from("s7.stain"), "Промолчать", "s7.search")

	g.beat("s7.pantry", bwForesterInterior,
		narr("На полках стоят пустые банки и лежат свёрнутые брезентовые полотнища. В пыли виден чистый прямоугольник — отсюда недавно убрали коробку.").hide(),
		thought("Кто-то забрал вещи после её смерти. Морган? Адриан? Почему никто не говорит, что именно здесь хранилось?").show(evelyn),
	)
	g.link(from("s7.pantry"), "Спросить, что здесь было", "s7.search",
		evelyn.say("Что здесь хранилось?"),
		adrian.say("Здесь были её записи."),
		evelyn.say("Были?"),
		adrian.say("Я не знаю, кто их забрал."),
	)
	g.link(from("s7.pantry"), "Не спрашивать", "s7.search")

	g.beat("s7.vision", bwForesterInterior,
		narr("Эвелин стоит между столом и пустой оправой.").show(evelyn),
		evelynSad.say("Она была здесь."),
		narr("Адриан не спрашивает, о ком речь.").show(adrian),
		evelynSad.say("В сером кардигане. На рукаве была кровь."),
		narr("Он медленно закрывает входную дверь, чтобы ветер не распахивал её.").show(adrian),
		adrian.say("Что ещё ты видела?"),
	)
	g.link(from("s7.vision"), "Рассказать всё", "s7.gift",
		evelyn.say("Синий свет. Стол. Она сказала: «Не отдавай». А потом ты отпустил мою руку, и всё исчезло."),
		narr("Адриан смотрит на её запястье.").show(adrian),
		adrian.say("Раньше с тобой такое случалось?"),
		evelyn.say("Нет."),
	)
	g.link(from("s7.vision"), "Скрыть часть", "s7.gift",
		evelyn.say("Сначала скажи, что произошло. Потом я решу, чем делиться."),
		adrian.say("Ты мне не доверяешь."),
		evelyn.say("У меня были причины?"),
		narr("Он принимает это без возражений.").show(adrian),
	)
	g.link(from("s7.vision"), "Обвинить его", "s7.gift",
		evelynWorried.say("Это ты сделал?"),
		adrian.say("Нет."),
		evelynWorried.say("Ты даже не спросил, что я имею в виду."),
		adrian.say("Потому что понимаю."),
		thought("Мне хотелось, чтобы он рассердился. Тогда было бы проще. А он выглядит так, будто я сказала то, что он сам повторяет себе каждую ночь."),
	)

	g.beat("s7.gift", bwForesterInterior,
		narr("Эвелин делает шаг к нему. Адриан отступает, прежде чем их руки успевают соприкоснуться.").show(adrian),
		evelyn.say("Вот опять."),
		adrian.say("Что?"),
		evelyn.say("Ты ни разу не прикоснулся ко мне после аварии."),
		narr("Молчание."),
		evelyn.say("Ты меня боишься?"),
		adrian.say("Нет."),
		evelyn.say("Тогда чего?"),
		narr("Он смотрит на её руку.").show(adrian),
		adrian.say("Того, что ты увидишь."),
		narr("Эвелин вспоминает парковку. Его пальцы на запястье. Вспышку. То, как быстро он её отпустил.").show(evelynWorried),
		thought("Это было не моё воспоминание."),
		evelynWorried.say("Я увидела твою память?"),
		adrian.say("Думаю, да."),
		evelyn.say("Ты думаешь?"),
		adrian.say("Обычно всё происходит наоборот."),
		narr("Она отступает от него.").show(evelynWorried),
		evelynWorried.say("Ты забираешь чужие воспоминания?"),
		narr("Он не отвечает сразу.").show(adrian),
		adrian.say("Могу."),
		narr("Слово остаётся между ними."),
	)
	g.link(from("s7.gift"), "Испугаться", "s7.offer",
		evelynWorried.say("Ты уже что-то забрал у меня?"),
		adrian.say("Нет."),
		evelynWorried.say("Я бы заметила?"),
		narr("Он медлит.").show(adrian),
		adrian.say("Не всегда."),
		narr("Эвелин отходит к двери.").show(evelynWorried),
		thought("Я не могу даже проверить, хватает ли мне всех моих воспоминаний. Как вообще задавать следующий вопрос?"),
	)
	g.link(from("s7.gift"), "Разозлиться", "s7.offer",
		evelyn.say("Поэтому тебе так удобно говорить, что мне показалось?"),
		narr("Адриан опускает голову.").show(adrian),
		adrian.say("Да. Это было удобно."),
		evelyn.say("Больше так не делай."),
		adrian.say("Не буду."),
		thought("Обещание. Пока просто обещание.").show(evelyn),
	)
	g.link(from("s7.gift"), "Попытаться разобраться", "s7.offer",
		evelyn.say("Но со мной не получилось?"),
		adrian.say("Я не пытался. Я удерживал тебя, чтобы машина не отбросила в сторону. А ты увидела то, чего видеть не должна была."),
		evelyn.say("Почему?"),
		adrian.say("Я не знаю."),
		thought("Впервые его «не знаю» звучит иначе. Но это ещё не доказательство.").show(evelyn),
	)

	g.beat("s7.offer", bwForesterInterior,
		evelyn.say("У моей бабушки ты что-нибудь забрал?"),
		adrian.say("Нет."),
		evelyn.say("Ты уже лгал мне."),
		adrian.say("Да."),
		narr("Он снимает перчатку, которую надел перед дорогой, и кладёт на стол.").show(adrian),
		narr("Протягивает открытую ладонь."),
		adrian.say("Поэтому я не прошу тебя поверить. Ты можешь посмотреть."),
	)
}

// --- Сцена 8. Чужая память ---

func sceneEight(g *storyGraph) {
	g.link(from("s7.offer"), "Посмотреть на его ладонь", "s8.choice")

	g.beat("s8.choice", bwForesterInterior,
		narr("В доме становится совсем темно. Фонарик лежит на столе, освещая руки Адриана снизу.").hide(),
		narr("За окном не видно другого берега."),
		thought("Несколько часов назад я хотела получить ответы. Теперь ответ стоит передо мной с протянутой рукой, а я боюсь, что после него не смогу вернуться к прежней жизни.").show(evelynWorried),
		narr("Адриан не приближается.").show(adrian),
	).titled("Чужая память")

	g.link(from("s8.choice"), "Согласиться сразу", "s8.memory",
		narr("Эвелин подходит.").show(evelyn),
		evelyn.say("Если я попрошу отпустить — отпустишь."),
		adrian.say("Сразу."),
		narr("Она кладёт ладонь в его руку."),
	)
	g.link(from("s8.choice"), "Сначала выяснить условия", "s8.terms")
	refuse := []scriptLine{
		evelyn.say("Нет. Ты расскажешь словами."),
		narr("Адриан медленно опускает руку.").show(adrian),
	}
	g.link(from("s8.choice"), "Отказаться", "s8.alt", refuse...)

	g.beat("s8.terms", bwForesterInterior,
		evelyn.say("Ты сможешь выбрать, что я увижу?"),
		adrian.say("Вчера не смог."),
		evelyn.say("Это больно?"),
		adrian.say("Не знаю."),
		evelyn.say("Ты вообще понимаешь, как это работает?"),
		adrian.say("С тобой — нет."),
		narr("Эвелин смотрит на его открытую ладонь.").show(evelynWorried),
		thought("Плохой ответ. Но хотя бы он не обещает, что всё будет хорошо."),
	)
	g.link(from("s8.terms"), "Прикоснуться", "s8.memory",
		narr("Она кладёт ладонь в его руку.").show(evelyn),
	)
	g.link(from("s8.terms"), "Отказаться", "s8.alt", refuse...)

	g.beat("s8.memory", bwForesterInterior,
		narr("Пальцы Адриана холодные.").hide(),
		narr("Сначала Эвелин ощущает только это."),
		narr("Потом исчезает звук дождя."),
		narr("Комната становится другой."),
		narr("Стол сдвинут к окну. На нём горит лампа. В металлической оправе на стене сияет целый синий камень."),
		narr("Эвелин пытается повернуть голову и понимает, что не управляет движением."),
		thought("Я не стою рядом с ним. Я смотрю его глазами."),
		narr("Под рёбрами тяжёлая боль. Чужое тело едва держится на ногах."),
		narr("Бабушка стоит возле стены. Серый кардиган порван на плече."),
		adrianInMemory.say("Отойди."),
		grandma.say("Если я отойду, оно пойдёт дальше."),
		narr("По камню расползается трещина."),
		narr("Пустая чашка на столе начинает дребезжать."),
		narr("За окном вода поднялась до самого крыльца."),
		thought("Дом стоит на склоне. Озеро не должно быть так близко."),
		adrianInMemory.say("Мы ещё можем успеть."),
		narr("Бабушка качает головой."),
		grandma.say("Уже нет."),
		narr("Она прижимает ладонь к камню."),
		narr("Свет становится ярче."),
		narr("На мгновение её лицо теряет всякое выражение."),
		grandma.say("Где…"),
		narr("Она смотрит вокруг."),
		adrianInMemory.say("Посмотри на меня."),
		narr("Бабушка поворачивается."),
		grandma.say("Я вас знаю?"),
		narr("Эвелин пытается заговорить."),
		thought("Бабушка. Это я. Я здесь."),
		narr("Но у воспоминания нет её голоса."),
		narr("Адриан подходит ближе."),
		adrianInMemory.say("Да. Ты меня знаешь."),
		narr("Бабушка пытается вспомнить. Не может."),
		narr("Её губы дрожат."),
		narr("Адриан обхватывает её плечи, собираясь оттащить от стены, но она удерживает его за рукав."),
		narr("На секунду в её глазах возвращается узнавание."),
		grandma.say("Не отдавай ей девочку."),
		adrianInMemory.say("Кому?"),
		narr("Бабушка будто не слышит."),
		grandma.say("Мою…"),
		narr("Она пытается произнести имя."),
		narr("Эвелин понимает какое."),
		thought("Не надо. Не пытайся. Я знаю, что ты меня помнишь."),
		narr("Камень раскалывается."),
		narr("Свет гаснет."),
		narr("Адриан успевает подхватить бабушку прежде, чем она падает."),
		narr("Эвелин вырывает руку.").show(evelynWorried),
		narr("Адриан сразу отпускает."),
		narr("Она отступает и ударяется о стол. Лампа падает, катится по полу."),
		narr("Тот же звук. Та же комната."),
		narr("Только бабушки больше нет.").show(evelynSad),
		thought("Это случилось здесь. Пока я жила своей жизнью, она была здесь."),
		narr("Адриан делает движение к ней.").show(adrian),
		evelynSad.say("Не надо."),
		narr("Он останавливается.").show(adrian),
	)
	g.link(from("s8.memory"), "Спросить о бабушке", "s8.reveal",
		evelynSad.say("Она забыла меня."),
		adrian.say("Не смогла вспомнить твоё имя."),
		evelynSad.say("Какая разница?"),
		narr("Он отвечает тихо:").show(adrian),
		adrian.say("До последнего она знала, что должна тебя защитить."),
		narr("Эвелин садится на стул.").show(evelynSad),
		thought("Я хочу поверить ему. Хотя бы в этом."),
	)
	g.link(from("s8.memory"), "Обвинить Адриана", "s8.reveal",
		evelynSad.say("Ты просто стоял рядом."),
		narr("Он не защищается.").show(adrian),
		adrian.say("Я пытался отвести её."),
		evelynSad.say("Недостаточно."),
		adrian.say("Знаю."),
		narr("Эвелин отворачивается.").show(evelynSad),
		thought("Я видела его боль. Чувствовала, что он едва держался. И всё равно хочу, чтобы виноват был он. Потому что он здесь, а того, что её убило, здесь нет."),
	)
	g.link(from("s8.memory"), "Держаться за факты", "s8.reveal",
		evelyn.say("Когда это было?"),
		adrian.say("В ночь её смерти."),
		evelyn.say("А потом её отвезли домой?"),
		adrian.say("Да."),
		evelyn.say("Кто?"),
		narr("Он делает короткую паузу.").show(adrian),
		adrian.say("Я."),
		thought("Хорошо. Ещё один факт. Пока я задаю вопросы, мне не нужно думать обо всём сразу.").show(evelyn),
	)

	g.beat("s8.reveal", bwForesterInterior,
		evelyn.say("Что она держала?"),
		narr("Адриан смотрит на пустую оправу.").show(adrian),
		adrian.say("Печать."),
		evelyn.say("От чего?"),
		narr("Над озером вспыхивает слабый синий свет.").hide(),
		narr("Адриан замечает его первым. Эвелин прослеживает его взгляд."),
		evelynWorried.say("Что под водой?"),
		adrian.say("То, что умеет забирать гораздо больше, чем мы."),
		evelyn.say("Она пыталась это остановить?"),
		adrian.say("Она остановила."),
		evelyn.say("Тогда почему это происходит снова?"),
		narr("Он молчит слишком долго.").show(adrian),
		thought("Потому что он знает ещё что-то. Всё это время знал.").show(evelynWorried),
		evelynWorried.say("Адриан."),
		adrian.say("Печать не разрушилась сама."),
		narr("Она медленно поворачивается к нему.").show(evelynWorried),
		adrian.say("Кто-то из моей семьи ей помог."),
	)
	g.link(from("s8.reveal"), "Затемнение", "s8.end")

	g.beat("s8.alt", bwForesterInterior,
		narr("Эвелин остаётся по другую сторону стола.").show(evelyn),
		evelyn.say("Ты предлагаешь мне залезть тебе в голову, потому что не можешь сказать правду вслух?"),
		adrian.say("Могу."),
		evelyn.say("Тогда скажи."),
		narr("Он садится на край стола.").show(adrian),
		adrian.say("Я был здесь в ночь её смерти."),
		thought("Я знала. Подпись в журнале, его лицо, этот дом. Почему от произнесённых слов всё равно становится хуже?").show(evelynSad),
		evelyn.say("Что произошло?"),
		adrian.say("Она удерживала печать. То, что находится под озером, пыталось выбраться. Она не позволила."),
		evelynSad.say("И умерла?"),
		narr("Он кивает.").show(adrian),
	)
	g.link(from("s8.alt"), "Потребовать доказательств", "s8.alt_end",
		evelyn.say("У меня пока только твои слова."),
		adrian.say("Да."),
		evelyn.say("Значит, я проверю каждое."),
		adrian.say("Проверяй."),
		thought("На этот раз я не позволю одному человеку объяснить мне всё так, как ему удобно.").show(evelyn),
	)
	g.link(from("s8.alt"), "Спросить о его роли", "s8.alt_end",
		evelyn.say("А ты что делал?"),
		adrian.say("Пытался её спасти."),
		evelyn.say("Получилось не очень."),
		narr("Он принимает эти слова, не отводя глаз.").show(adrian),
		thought("Он даже не оправдывается. Это может быть вина. А может быть то, что я хочу принять за вину.").show(evelyn),
	)
	g.link(from("s8.alt"), "Спросить о последних словах", "s8.alt_end",
		evelynSad.say("Она что-нибудь сказала?"),
		narr("Адриан долго смотрит на пустую оправу.").show(adrian),
		adrian.say("Просила не отдавать тебя."),
		evelynSad.say("Кому?"),
		adrian.say("Она не успела сказать."),
		narr("Эвелин обхватывает себя руками.").show(evelynSad),
		thought("Мне хочется, чтобы она звала меня. И страшно, что она действительно звала."),
	)

	g.beat("s8.alt_end", bwForesterInterior,
		narr("За окном над водой вспыхивает синий свет.").hide(),
		narr("Эвелин замечает, как резко Адриан встаёт.").show(adrian),
		evelynWorried.say("Ты сказал, она это остановила."),
		adrian.say("Остановила."),
		evelynWorried.say("Тогда что сейчас происходит?"),
		narr("Он смотрит на озеро.").show(adrian),
		adrian.say("То же, что произошло в ту ночь."),
		evelynWorried.say("Почему?"),
		narr("Адриан поворачивается к ней.").show(adrian),
		adrian.say("Потому что печать не разрушилась сама. Кто-то из моей семьи ей помог."),
		thought("Я ещё не знаю, верю ли ему. Но теперь точно знаю, что не уеду.").show(evelyn),
	)
	g.link(from("s8.alt_end"), "Затемнение", "s8.end")

	g.beat("s8.end", bwForesterExterior,
		narr("Конец первой серии.").hide(),
	)
}
