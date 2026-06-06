# Wolfy

Wolfy 是一个面向 Bilibili 直播间的舞萌 DX 点歌工具。它会连接 Bilibili 直播开放平台，解析弹幕里的点歌指令，维护本地点歌队列，并通过内置静态页面展示当前队列、歌曲封面、曲目信息和操作反馈。

## 中文说明

### 功能

- 从 Bilibili 直播开放平台接收弹幕任务。
- 支持点歌、换歌、换谱、删除当前点歌。
- 基于舞萌曲库 XML 和 alias 数据做歌曲模糊匹配。
- 内置 Web 页面和 JSON API，用于展示当前点歌队列与临时消息。
- 点歌队列会写入 `runtime/tickets.checkpoint.json`，消息会写入 `runtime/messages.checkpoint.json`。
- 每人最多同时保留 3 首未完成点歌。

### 环境要求

- Go 1.24.2 或更高版本。
- 可用的舞萌歌曲数据包目录，目录内需要包含 `Music.xml` 文件。
- Bilibili 直播开放平台配置：
  - `APP_ID`
  - `ANCHOR_CODE`
  - `BILIBILI_AK_ID`
  - `BILIBILI_AK_SECRET`

如果本地没有 `BILIBILI_AK_ID` 和 `BILIBILI_AK_SECRET`，程序会尝试使用远程签名服务。

### 配置

运行本地服务前设置环境变量：

```sh
export APP_ID="your_app_id"
export ANCHOR_CODE="your_anchor_code"
export BILIBILI_AK_ID="your_access_key_id"
export BILIBILI_AK_SECRET="your_access_key_secret"
export SONG_PACKAGE_PATH="/path/to/maimai/package"
export ALIAS_FILE_PATH="/path/to/alias.json"
```

`ALIAS_FILE_PATH` 是可选项。如果提供本地 alias 文件，程序会优先读取本地文件；否则会尝试从远程接口获取 alias 数据。

### 运行

启动本地服务：

```sh
go run .
```

服务启动后会监听：

- Web 页面：`http://localhost:41377/static/`
- API：`http://localhost:41377/api/...`

构建 Windows 可执行文件：

```sh
GOOS=windows GOARCH=amd64 go build -o main.exe .
```

启动远程签名服务：

```sh
go run -tags remote .
```

远程签名服务监听 `41376` 端口，并提供：

```text
POST /sign
```

### 弹幕指令

当前支持的弹幕关键词：

- `点歌 <歌名>`：新增点歌。
- `换歌 <编号>`：切换到当前关键词下一个匹配结果。
- `换谱 <编号>`：切换谱面难度。
- `删除 <编号>`：删除点歌。

### HTTP API

```text
GET /api/tickets
GET /api/messages
GET /api/event/:caller/:event/:content
```

前端事件包括：

- `pick`
- `click_cover_info`
- `click_genre_info`
- `click_song_info`
- `click_creator`

### 测试

运行全部测试：

```sh
env GOCACHE=/tmp/wolfy-go-build go test ./...
```

验证远程签名入口：

```sh
env GOCACHE=/tmp/wolfy-go-build go test -tags remote .
```

### 前端静态资源

仓库中的 `static/` 目录包含服务直接托管的前端资源。如果需要从相邻的 `wolfy_web` 项目重新构建前端资源，可以参考：

```sh
scripts/gao.sh
```

---

## English

Wolfy is a Maimai DX song-request tool for Bilibili live rooms. It connects to the Bilibili Live Open Platform, parses song-request commands from live chat messages, maintains a local request queue, and serves a built-in static web page for displaying the queue, covers, song metadata, and operation messages.

### Features

- Receives live-chat tasks from the Bilibili Live Open Platform.
- Supports song request, rerolling the matched song, changing chart difficulty, and deleting requests.
- Uses Maimai XML song data and alias data for fuzzy song matching.
- Serves a built-in web UI and JSON API for the current queue and transient messages.
- Persists tickets to `runtime/tickets.checkpoint.json` and messages to `runtime/messages.checkpoint.json`.
- Limits each user to at most 3 pending song requests.

### Requirements

- Go 1.24.2 or newer.
- A Maimai song package directory containing `Music.xml` files.
- Bilibili Live Open Platform configuration:
  - `APP_ID`
  - `ANCHOR_CODE`
  - `BILIBILI_AK_ID`
  - `BILIBILI_AK_SECRET`

If `BILIBILI_AK_ID` and `BILIBILI_AK_SECRET` are not available locally, the app will try to use the remote signing service.

### Configuration

Set environment variables before running the local service:

```sh
export APP_ID="your_app_id"
export ANCHOR_CODE="your_anchor_code"
export BILIBILI_AK_ID="your_access_key_id"
export BILIBILI_AK_SECRET="your_access_key_secret"
export SONG_PACKAGE_PATH="/path/to/maimai/package"
export ALIAS_FILE_PATH="/path/to/alias.json"
```

`ALIAS_FILE_PATH` is optional. If a local alias file is provided, Wolfy reads it first; otherwise it tries to fetch alias data from the remote endpoint.

### Run

Start the local service:

```sh
go run .
```

The service listens on:

- Web UI: `http://localhost:41377/static/`
- API: `http://localhost:41377/api/...`

Build a Windows executable:

```sh
GOOS=windows GOARCH=amd64 go build -o main.exe .
```

Start the remote signing service:

```sh
go run -tags remote .
```

The remote signing service listens on port `41376` and exposes:

```text
POST /sign
```

### Live Chat Commands

Supported live-chat keywords:

- `点歌 <song name>`: add a song request.
- `换歌 <index>`: switch to the next matched song for the same keyword.
- `换谱 <index>`: switch chart difficulty.
- `删除 <index>`: remove a song request.

### HTTP API

```text
GET /api/tickets
GET /api/messages
GET /api/event/:caller/:event/:content
```

Frontend events:

- `pick`
- `click_cover_info`
- `click_genre_info`
- `click_song_info`
- `click_creator`

### Tests

Run the full test suite:

```sh
env GOCACHE=/tmp/wolfy-go-build go test ./...
```

Verify the remote-signing entrypoint:

```sh
env GOCACHE=/tmp/wolfy-go-build go test -tags remote .
```

### Frontend Static Assets

The `static/` directory contains the frontend assets served by Wolfy. To rebuild them from the adjacent `wolfy_web` project, see:

```sh
scripts/gao.sh
```
