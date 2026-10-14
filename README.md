# query

Унифицированная Go-библиотека для опроса игровых серверов: Minecraft Java (Modern и Legacy), Minecraft Bedrock, Source/GoldSource (A2S). Все протоколы возвращают один и тот же тип `Server`, что упрощает работу с разнородными серверами.

## Возможности

- **Единый API** для четырёх протоколов: `ServerMinecraftModern`, `ServerMinecraftLegacy`, `ServerMinecraftBedrock`, `ServerSource`.
- **Авто-режим** — параллельно пробует Java, Bedrock и Source, возвращает первый успешный ответ.
- **MOTD в трёх форматах** — `raw` (с §-кодами), `clean` (чистый текст), `html` (с `<span>` и экранированием).
- **Блокировка localhost** на уровне резолва, включая защиту от DNS-байпаса.
- **Хуки** для метрик и логирования.
- **Кастомный resolver** для тестов и нестандартного DNS.
- **Пакетные запросы** с дедупликацией и ограничением параллелизма.
- **Безопасен к отмене** — `context.Context` обрывает ожидание.
- **Ноль внешних зависимостей** кроме `mcutil` и `woozymasta/a2s`.

## Установка

```bash
go get github.com/NotRaelc/query
```

Требуется Go 1.22+.

## Быстрый старт

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/NotRaelc/query"
)

func main() {
	resp := query.QueryServer(
		context.Background(),
		"play.example.com",
		5*time.Second,
	)

	if resp.Error != nil {
		log.Fatal(resp.Error)
	}

	s := resp.Payload
	fmt.Printf("%s | %d/%d\n", s.Name, s.OnlinePlayers, s.MaxPlayers)
}
```

## Поддерживаемые протоколы

| Константа | Транспорт | Порт по умолчанию | Описание |
|---|---|---|---|
| `ServerMinecraftModern` | TCP | 25565 | Java 1.7+, JSON-ответ |
| `ServerMinecraftLegacy` | TCP | 25565 | Java до 1.7, строковый ответ |
| `ServerMinecraftBedrock` | UDP (RakNet) | 19132 | Bedrock, Unconnected Ping |
| `ServerSource` | UDP (A2S) | — | Source/GoldSource (CS, TF2, Rust, DayZ, Arma 3 и т.д.) |
| `ServerAuto` | все сразу | — | Параллельный probe трёх транспортов |

## API

### `QueryServer`

```go
func QueryServer(
	ctx context.Context,
	addrStr string,
	timeout time.Duration,
	st ...ServerType,
) Response
```

Основная точка входа. `addrStr` — `host` или `host:port`. Если порт не указан, подставляется `DefaultMinecraftPort` (25565). Если `st` не задан — `ServerAuto`.

Возвращает `Response`. `Response.Error == nil` означает успех.

### `QueryServerOpts`

```go
func QueryServerOpts(ctx context.Context, addrStr string, opts Options) Response
```

Полная версия с `Options`. Используйте её, если нужны хуки, кастомный resolver, отключение иконок или игроков.

### `QueryMultipleServers`

```go
func QueryMultipleServers(
	ctx context.Context,
	addrStrings []string,
	timeout time.Duration,
	st ...ServerType,
) []Response
```

Пакетный вариант. Все адреса опрашиваются параллельно, порядок результата совпадает с порядком входа.

### `QueryMultipleServersOpts`

```go
func QueryMultipleServersOpts(ctx context.Context, addrStrings []string, opts Options) []Response
```

Пакетный вариант с `Options`. Дубликаты (по `host:port:kind`) опрашиваются один раз, результат раздаётся всем слотам.

### `Options`

```go
type Options struct {
	Timeout        time.Duration
	Type           ServerType
	NoIcon         bool
	NoPlayers      bool
	MaxConcurrent  int
	Hooks          Hooks
	Resolver       Resolver
}
```

| Поле | Назначение |
|---|---|
| `Timeout` | Верхняя граница запроса. `0` → `DefaultTimeout` (5s). |
| `Type` | Протокол. `0` → `ServerAuto`. |
| `NoIcon` | Не включать `Icon` в результат. Экономит трафик. |
| `NoPlayers` | Не включать список игроков. Экономит трафик. |
| `MaxConcurrent` | Лимит параллельных запросов в batch. `0` → `DefaultMaxConcurrent` (20). |
| `Hooks` | Callback-и для метрик. |
| `Resolver` | Кастомный DNS. `nil` → системный резолвер. |

### `Hooks`

```go
type Hooks struct {
	OnAttempt func(host string, proto ServerType)
	OnResult  func(host string, proto ServerType, err error, d time.Duration)
}
```

`OnAttempt` вызывается перед каждым запросом, `OnResult` — после (с успехом или ошибкой).

```go
opts := query.Options{
	Timeout: 5 * time.Second,
	Hooks: query.Hooks{
		OnResult: func(host string, proto query.ServerType, err error, d time.Duration) {
			metrics.ObserveDuration(host, proto.String(), d)
			if err != nil {
				metrics.IncError(host, proto.String())
			}
		},
	},
}
resp := query.QueryServerOpts(ctx, "play.example.com", opts)
```

### `Resolver`

```go
type Resolver func(ctx context.Context, host string) ([]netip.Addr, error)
```

Возвращает адреса в порядке приоритета: сначала IPv4, потом IPv6. Полезно для тестов и нестандартного DNS.

### `Response`

```go
type Response struct {
	Error   error  `json:"-"`
	Payload Server `json:"payload"`
}
```

`Response` реализует `MarshalJSON`: при ошибке `Error` уходит строкой, при успехе поле `error` отсутствует.

### `Server`

```go
type Server struct {
	GameKind      string   // "minecraft-java" | "minecraft-legacy" | "minecraft-bedrock" | "source"
	Online        bool
	Name          string
	Map           string   // настоящее имя карты; для Minecraft ping — пустое
	GameID        int64
	Protocol      int64
	Versions      []string

	IP            string
	Port          uint16
	Hostname      string   // заполняется только если input был доменом

	Players       []Player
	OnlinePlayers int64
	MaxPlayers    int64

	Icon          string
	Software      string
	Gamemode      string
	GamemodeID    int64
	ServerID      string
	EulaBlocked   bool
	MOTD          *MOTD
	Plugins       []Plugin
	Mods          []Mod
	Info          *MOTD
	Keywords      any      // структура из woozymasta/a2s/pkg/keywords для Arma 3 / DayZ
	Debug         *DebugInfo
}
```

Поля, которые не отдаёт конкретный протокол, остаются с нулевым значением. Слайсы всегда не-`nil` (`[]`, не `null`), чтобы фронту не приходилось проверять.

### `MOTD`

```go
type MOTD struct {
	Raw   []string `json:"raw"`
	Clean []string `json:"clean"`
	HTML  []string `json:"html"`
}
```

Каждый элемент — одна строка. `Raw` содержит §-коды, `Clean` — только текст, `HTML` — `<span>` с inline-цветами.

### `Player`

```go
type Player struct {
	Name  string
	ID    string   // UUID для Java, "-1" для Source
	Time  int64    // секунды; -1 если неизвестно
	Score int32    // только Source
	Index uint8
}
```

### `ProtocolError`

Ошибки запросов оборачиваются в `*ProtocolError`, что позволяет узнать, какой именно протокол упал:

```go
var pe *query.ProtocolError
if errors.As(resp.Error, &pe) {
	log.Printf("protocol %s failed: %v", pe.Proto, pe.Err)
}
```

### Sentinel-ошибки

- `ErrTimeout` — общий таймаут запроса.
- `ErrBlockedHost` — адрес указывает на localhost или loopback.
- `ErrProtocolFailed` — сервер ответил, но не тем, что ожидалось.

## Блокировка localhost

Все запросы проходят проверку `IsBlockedHost` до сетевого взаимодействия. Блокируются:

- Строка `localhost` (регистронезависимо).
- Любой loopback-адрес (`127.0.0.0/8`, `::1`).
- Любой unspecified-адрес (`0.0.0.0`, `::`).

**DNS-байпас невозможен.** Если домен резолвится в loopback — запрос отклоняется. Резолв занимает один запрос к системному DNS; при ошибке резолва запрос не блокируется и идёт дальше.

## Примеры

### Batch-запрос

```go
addrs := []string{
	"mc.hypixel.net",
	"play.cubecraft.net",
	"play.hivebedrock.network:19132",
	"chi-1.us.uncletopia.com:27015",
}

responses := query.QueryMultipleServers(
	context.Background(),
	addrs,
	8*time.Second,
)

for i, r := range responses {
	if r.Error != nil {
		fmt.Printf("%s: %v\n", addrs[i], r.Error)
		continue
	}
	fmt.Printf("%s: %s %d/%d\n",
		addrs[i], r.Payload.Name, r.Payload.OnlinePlayers, r.Payload.MaxPlayers)
}
```

### Разные протоколы в одном батче

`QueryMultipleServers` использует один `ServerType` на весь батч. Если нужны разные — вызывайте `QueryServerOpts` параллельно:

```go
targets := []struct {
	addr string
	kind query.ServerType
}{
	{"mc.hypixel.net", query.ServerMinecraftModern},
	{"play.hivebedrock.network:19132", query.ServerMinecraftBedrock},
	{"chi-1.us.uncletopia.com:27015", query.ServerSource},
}

responses := make([]query.Response, len(targets))
var wg sync.WaitGroup
sem := make(chan struct{}, 5)

for i, t := range targets {
	wg.Add(1)
	go func(i int, addr string, kind query.ServerType) {
		defer wg.Done()
		sem <- struct{}{}
		defer func() { <-sem }()

		responses[i] = query.QueryServerOpts(ctx, addr, query.Options{
			Timeout: 8 * time.Second,
			Type:    kind,
		})
	}(i, t.addr, t.kind)
}
wg.Wait()
```

### HTTP-обработчик

```go
func handleServers(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("addr")
	if addr == "" {
		http.Error(w, "addr required", http.StatusBadRequest)
		return
	}

	resp := query.QueryServer(
		r.Context(),
		addr,
		5*time.Second,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
```

`r.Context()` обрывает запрос, если клиент отвалился. Для кеша передавайте `context.Background()` с явным таймаутом — тогда заполнение кеша не зависит от пользователя.

### Тестирование без сети

```go
opts := query.Options{
	Timeout: time.Second,
	Resolver: func(_ context.Context, _ string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
	},
}
resp := query.QueryServerOpts(ctx, "example.com", opts)
```

## Формат JSON

Успешный ответ:

```json
{
  "payload": {
    "game_kind": "minecraft-java",
    "online": true,
    "name": "CubeVerse",
    "map": "",
    "game_id": -1,
    "protocol": 765,
    "versions": ["1.20.4"],
    "ip": "cubeverse.su",
    "port": 25565,
    "hostname": "cubeverse.su",
    "players": [
      {"name": "Player1", "id": "uuid...", "time": -1, "index": 0}
    ],
    "online_players": 42,
    "max_players": 2000,
    "motd": {
      "raw": ["§aWelcome"],
      "clean": ["Welcome"],
      "html": ["<span style=\"color: #55FF55\">Welcome</span>"]
    }
  }
}
```

Ответ с ошибкой:

```json
{
  "error": "modern: EOF"
}
```

## Поведение и гарантии

- **Дубликаты в batch-запросе.** Схлопываются по `(host, port, kind)`. Один запрос — один результат, разданный всем совпавшим слотам.
- **Порядок результатов.** Всегда совпадает с порядком входных адресов.
- **Отмена через context.** `ctx` возвращает ошибку из `QueryServer` досрочно. Внутри `mcutil` сетевая операция продолжает жить до `opts.Timeout` — это ограничение `mcutil`, не библиотеки. Устанавливайте `opts.Timeout` разумным (3–10 s), чтобы не держать сокеты лишний раз.
- **Порты по умолчанию.** Для Java — 25565, для Bedrock — 19132. Если порт указан явно — он используется для всех протоколов.
- **Список игроков.** Для Java и Bedrock он неполный — сервер отдаёт только sample. Для Source — полный список.
- **MOTD для Source** — это `info.Name`, разбитый на строки. Для Bedrock — первая и вторая строки MOTD, разделённые `\n`.

## Ограничения

- **Зависимость от `mcutil`.** Сетевые операции внутри неё не отменяются по `context` — только по `opts.Timeout`. См. раздел «Поведение и гарантии».
- **Bedrock по умолчанию на 25565.** В текущей версии `ServerAuto` не подставляет 19132 для Bedrock-проб. Для надёжного Bedrock-запроса указывайте порт явно.
- **Minecraft не отдаёт карту через ping.** Поле `Map` для Java/Bedrock всегда пустое. Для Source — имя карты, как в протоколе.
- **Sample-as-MOTD.** Некоторые BungeeCord/Velocity-серверы кладут в `Players.Sample` рекламные строки. Библиотека их не фильтрует — делайте это на стороне потребителя.

## Тестирование

```bash
# Все unit-тесты
go test -short ./...

# Включая live-тесты с публичными серверами
go test ./...

# Fuzzing
go test -fuzz=FuzzSplitHostPort -fuzztime=60s
go test -fuzz=FuzzOptionsIsBlocked -fuzztime=60s

# Бенчмарки
go test -bench=. -benchmem -run=^$ ./...

# Покрытие
go test -short -coverprofile=cover.out ./...
go tool cover -html=cover.out
```

## Лицензия

MIT. См. `LICENSE` в репозитории.

## Ссылки

- Спецификация Java SLP: https://wiki.vg/Server_List_Ping
- Спецификация A2S: https://developer.valvesoftware.com/wiki/Server_queries
- RakNet (Bedrock): https://wiki.vg/Raknet_Protocol
- Исходники `mcutil`: https://github.com/mcstatus-io/mcutil
- Исходники `woozymasta/a2s`: https://github.com/WoozyMasta/a2s