namespace go wolfy.api

/*
 * Wolfy HTTP API contract
 *
 * 当前后端不是 Thrift RPC 服务；这里用 Thrift IDL 描述 JSON 请求、
 * JSON 响应、路径参数和字段含义，方便按统一契约开发前端功能。
 *
 * 本地 HTTP 服务：
 *   - 默认地址：http://localhost:41377
 *   - 前缀：/api
 *
 * 远程弹幕服务 remote build：
 *   - 默认地址：http://localhost:41376
 *   - 仅在 `go run -tags remote .` 启动
 *   - 负责 Bilibili AK/SK、本场 app lifecycle、弹幕 websocket 和命令解析
 *
 * 通用错误格式：
 *   - {"msg": "<错误信息>"}
 */

typedef string AnchorCode
typedef string ComponentName
typedef string ComponentStatusText
typedef string ComponentEventTypeText
typedef string FrontendEvent
typedef string GameID
typedef string StatusText

enum ComponentStatus {
  WAITING = 1,
  RUNNING = 2,
  ERROR = 3,
  RESTARTING = 4,
}

enum FrontendEventType {
  PICK = 1,
  CLICK_COVER_INFO = 2,
  CLICK_GENRE_INFO = 3,
  CLICK_SONG_INFO = 4,
  CLICK_CREATOR = 5,
}

struct ApiError {
  1: required string msg,
}

struct Task {
  1: required string command,
  2: required string caller,
  3: required string content,
  4: required i64 index,
}

struct StartGameRequest {
  1: required i64 app_id,
  2: required AnchorCode anchor_code,
  3: optional bool force,
}

struct StopGameRequest {
  1: optional string reason,
}

struct AnchorInfo {
  1: required i64 room_id,
  2: required string uname,
  3: required string uface,
  4: required i64 uid,
  5: required string open_id,
}

struct GameSession {
  1: required AnchorCode anchor_code,
  2: required i64 app_id,
  3: required GameID game_id,
  4: required StatusText status,
  5: required string started_at,
  6: required string last_heartbeat_at,
  7: required i64 last_seq,
  8: optional AnchorInfo anchor,
  9: optional string error,
}

struct StartGameResponse {
  1: required GameSession data,
}

struct StopGameResponse {
  1: required GameSession data,
}

struct DanmuEvent {
  1: required i64 seq,
  2: required AnchorCode anchor_code,
  3: required string msg_id,
  4: required i64 room_id,
  5: required string caller,
  6: required i64 uid,
  7: required string uface,
  8: required string message,
  9: required i64 timestamp,
  10: required string received_at,
  11: optional string raw_cmd,
  12: optional Task task,
}

struct PullDanmuResponse {
  1: required list<DanmuEvent> events,
  2: required i64 next_seq,
  3: required bool has_more,
}

struct RemoteDanmuConfig {
  1: required string remote_base_url,
  2: required i64 app_id,
  3: required AnchorCode anchor_code,
}

struct UpdateRemoteDanmuConfigRequest {
  1: required RemoteDanmuConfig config,
}

struct LocalDanmuStatus {
  1: required StatusText status,
  2: required RemoteDanmuConfig config,
  3: required i64 last_seq,
  4: optional string error,
}

struct LocalDanmuStatusResponse {
  1: required LocalDanmuStatus data,
}

struct Message {
  1: required string content,
}

struct ComponentEvent {
  1: required string time,
  2: required ComponentName component,
  3: required ComponentEventTypeText type,
  4: required string code_location,
  5: required string message,
}

struct ComponentEventTypeInfo {
  1: required ComponentEventTypeText type,
  2: required string description,
}

struct ComponentSnapshot {
  1: required ComponentName name,
  2: required ComponentStatusText status,
  3: required string error,
  4: required map<string, string> params,
  5: required list<ComponentEvent> events,
}

struct TicketItem {
  1: required string title,
  2: required string keyword,
  3: required string creator,
  4: required string image,
  5: required string cover_info,
  6: required string genre_info,
  7: required string song_info,
}

struct GetTicketsData {
  1: required list<TicketItem> tickets,
}

struct GetTicketsResponse {
  1: required GetTicketsData data,
}

struct GetMessagesData {
  1: required list<Message> messages,
}

struct GetMessagesResponse {
  1: required GetMessagesData data,
}

struct EventResponse {
  1: required string data,
}

struct ComponentsResponse {
  1: required list<ComponentSnapshot> data,
}

struct ComponentEventTypesData {
  1: required list<ComponentEventTypeInfo> types,
}

struct ComponentEventTypesResponse {
  1: required ComponentEventTypesData data,
}

struct UpdateComponentParamsRequest {
  1: required map<string, string> params,
}

struct UpdateComponentParamsResponse {
  1: required ComponentSnapshot data,
}

struct RestartComponentResponse {
  1: required ComponentSnapshot data,
}

struct StopComponentResponse {
  1: required ComponentSnapshot data,
}

/*
 * 远程弹幕 HTTP API。
 *
 * 命令解析：
 *   - "点歌 <歌名>" -> task.command="pick", task.content=<歌名>, task.index=-1
 *   - "换歌 <序号>" -> task.command="next_rank", task.content="", task.index=<0-based>
 *   - "换谱 <序号>" -> task.command="next_level", task.content="", task.index=<0-based>
 *   - "删除 <序号>" -> task.command="finish", task.content="", task.index=<0-based>
 */
service WolfyRemoteDanmuHTTPAPI {
  /*
   * GET /healthz
   */
  string health();

  /*
   * POST /openapi/games
   */
  StartGameResponse startGame(1: StartGameRequest request);

  /*
   * GET /openapi/games/:anchor_code
   */
  StartGameResponse getGame(1: AnchorCode anchor_code);

  /*
   * DELETE /openapi/games/:anchor_code
   */
  StopGameResponse stopGame(
    1: AnchorCode anchor_code,
    2: StopGameRequest request,
  );

  /*
   * GET /openapi/games/:anchor_code/danmu?after_seq=0&limit=100&wait_ms=30000
   */
  PullDanmuResponse pullDanmu(
    1: AnchorCode anchor_code,
    2: i64 after_seq,
    3: i32 limit,
    4: i32 wait_ms,
  );
}

/*
 * 本地 Wolfy HTTP API。
 */
service WolfyLocalHTTPAPI {
  /*
   * GET /static/*filepath
   */
  binary getStaticAsset(1: string filepath);

  /*
   * GET /api/tickets
   */
  GetTicketsResponse getTickets();

  /*
   * GET /api/messages
   */
  GetMessagesResponse getMessages();

  /*
   * GET /api/event/:caller/:event/:content
   */
  EventResponse emitFrontendEvent(
    1: string caller,
    2: FrontendEvent event,
    3: string content,
  );

  /*
   * GET /api/sysinfo
   */
  ComponentsResponse getSysInfo();

  /*
   * GET /api/components
   */
  ComponentsResponse getComponents();

  /*
   * GET /api/component-event-types
   */
  ComponentEventTypesResponse getComponentEventTypes();

  /*
   * PATCH /api/components/:name/params
   */
  UpdateComponentParamsResponse updateComponentParams(
    1: ComponentName name,
    2: UpdateComponentParamsRequest request,
  );

  /*
   * POST /api/components/:name/restart
   */
  RestartComponentResponse restartComponent(1: ComponentName name);

  /*
   * POST /api/components/:name/stop
   */
  StopComponentResponse stopComponent(1: ComponentName name);

  /*
   * GET /api/danmu
   */
  LocalDanmuStatusResponse getDanmuStatus();

  /*
   * PATCH /api/danmu
   */
  LocalDanmuStatusResponse updateDanmuConfig(1: UpdateRemoteDanmuConfigRequest request);

  /*
   * POST /api/danmu/start
   */
  LocalDanmuStatusResponse startDanmu();

  /*
   * POST /api/danmu/stop
   */
  LocalDanmuStatusResponse stopDanmu();
}
