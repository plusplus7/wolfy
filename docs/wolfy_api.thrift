namespace go wolfy.api

/*
 * Wolfy HTTP API contract
 *
 * 这份文件用于给前端同学阅读当前后端支持的 HTTP API。
 * 当前后端不是 Thrift RPC 服务；这里用 Thrift IDL 描述 JSON 请求、JSON 响应、
 * 路径参数和字段含义，方便按统一契约开发前端功能。
 *
 * 本地 HTTP 服务：
 *   - 默认地址：http://localhost:41377
 *   - 前缀：/api
 *
 * 远程签名服务 remote build：
 *   - 默认地址：http://localhost:41376
 *   - 仅在 `go run -tags remote .` 启动时提供 /sign
 *
 * 通用错误格式：
 *   - 大多数 /api 接口失败时返回：{"msg": "<错误信息>"}
 *   - /sign 失败时返回：{"message": "<错误信息>"}
 */

typedef string ComponentName
typedef string ComponentStatusText
typedef string ComponentEventTypeText
typedef string FrontendEvent

/*
 * 组件状态枚举说明。
 *
 * 实际 JSON 字段 `status` 当前返回的是字符串，而不是数字：
 *   - "waiting"：等待启动。通常表示必填参数为空，或依赖组件尚未运行。
 *   - "running"：运行中。组件已经成功启动并可提供能力。
 *   - "error"：运行出错。`error` 字段会包含最近一次错误信息。
 *   - "restarting"：重启中。组件正在停止旧实例或启动新实例。
 */
enum ComponentStatus {
  WAITING = 1,
  RUNNING = 2,
  ERROR = 3,
  RESTARTING = 4,
}

/*
 * 前端事件枚举说明。
 *
 * 实际路径参数 `event` 当前使用字符串：
 *   - "pick"：主播/前端手动点歌，content 为歌曲关键词。
 *   - "click_cover_info"：点击封面左上角类型信息，语义为删除指定点歌。
 *   - "click_genre_info"：点击曲风信息，语义为切换到同关键词的下一个匹配歌曲。
 *   - "click_song_info"：点击谱面信息，语义为切换谱面难度。
 *   - "click_creator"：点击点歌人，语义为用 content 再次点歌。
 *
 * 对 click_* 类事件，content 通常是前端列表里的 0-based index 字符串。
 * 对 pick/click_creator，content 是歌曲关键词。
 */
enum FrontendEventType {
  PICK = 1,
  CLICK_COVER_INFO = 2,
  CLICK_GENRE_INFO = 3,
  CLICK_SONG_INFO = 4,
  CLICK_CREATOR = 5,
}

/*
 * 组件事件类型枚举说明。
 *
 * 实际 JSON 字段 `type` 当前返回的是字符串，而不是数字：
 *   - "component.status_changed"：组件状态发生变化。
 *   - "component.params_updated"：组件参数被更新。
 *   - "songs.storage_loaded"：songs 成功加载歌曲库。
 *   - "songs.storage_load_failed"：songs 加载歌曲库失败。
 *   - "tickets.task_received"：tickets 收到 task。
 *   - "tickets.task_completed"：tickets 完成 task。
 *   - "tickets.task_failed"：tickets 处理 task 失败。
 *   - "messages.message_received"：messages 收到 message。
 *   - "messages.message_stored"：messages 已存储 message。
 *   - "danmu.listener_started"：开放平台 danmu 监听启动成功。
 *   - "danmu.listener_failed"：开放平台 danmu 监听启动失败。
 *   - "danmu.danmu_received"：开放平台 danmu 收到弹幕。
 *   - "danmu.task_delivered"：开放平台 danmu 向 tickets 投递 task。
 *   - "blivedm.client_started"：blivedm client 启动成功。
 *   - "blivedm.client_failed"：blivedm client 启动失败。
 *   - "blivedm.danmu_received"：blivedm 收到弹幕。
 *   - "blivedm.task_delivered"：blivedm 向 tickets 投递 task。
 */
enum ComponentEventTypeEnum {
  COMPONENT_STATUS_CHANGED = 1,
  COMPONENT_PARAMS_UPDATED = 2,
  SONGS_STORAGE_LOADED = 3,
  SONGS_STORAGE_LOAD_FAILED = 4,
  TICKETS_TASK_RECEIVED = 5,
  TICKETS_TASK_COMPLETED = 6,
  TICKETS_TASK_FAILED = 7,
  MESSAGES_MESSAGE_RECEIVED = 8,
  MESSAGES_MESSAGE_STORED = 9,
  DANMU_LISTENER_STARTED = 10,
  DANMU_LISTENER_FAILED = 11,
  DANMU_DANMU_RECEIVED = 12,
  DANMU_TASK_DELIVERED = 13,
  BLIVEDM_CLIENT_STARTED = 14,
  BLIVEDM_CLIENT_FAILED = 15,
  BLIVEDM_DANMU_RECEIVED = 16,
  BLIVEDM_TASK_DELIVERED = 17,
}

/*
 * HTTP 错误响应。
 *
 * 字段：
 *   - msg：/api 接口的错误信息，例如参数不合法、组件不存在、tickets 未运行。
 */
struct ApiError {
  1: required string msg,
}

/*
 * /sign 错误响应。
 *
 * 字段：
 *   - message：远程签名服务的错误信息。
 */
struct SignError {
  1: required string message,
}

/*
 * 一条消息。
 *
 * 字段：
 *   - content：消息内容。后端约定以 "inf " 或 "err " 开头：
 *       - "inf " 表示成功/提示消息。
 *       - "err " 表示错误消息。
 *     当前前端会去掉前 4 个字符展示正文。
 */
struct Message {
  1: required string content,
}

/*
 * 一条组件内部事件。
 *
 * 字段：
 *   - time：事件发生时间，RFC3339Nano 字符串。
 *   - component：组件名。
 *   - type：事件类型字符串，见 ComponentEventTypeEnum 注释。
 *   - code_location：记录事件的代码位置，格式为 repo 相对路径加行号。
 *   - message：事件补充信息。
 */
struct ComponentEvent {
  1: required string time,
  2: required ComponentName component,
  3: required ComponentEventTypeText type,
  4: required string code_location,
  5: required string message,
}

/*
 * 组件事件类型说明。
 *
 * 字段：
 *   - type：事件类型字符串。
 *   - description：事件含义说明。
 */
struct ComponentEventTypeInfo {
  1: required ComponentEventTypeText type,
  2: required string description,
}

/*
 * 点歌队列中的一个展示项。
 *
 * 字段：
 *   - title：匹配到的歌曲标题；空槽位为占位文案。
 *   - keyword：用户输入或弹幕解析出的点歌关键词。
 *   - creator：点歌人名称；空槽位为 "-"。
 *   - image：歌曲封面 URL。
 *   - cover_info：谱面类型，常见值为 "std"、"dx"、"宴"；点击该字段触发删除。
 *   - genre_info：歌曲分类，例如 "舞萌"、"流行&动漫"、"东方Project"。
 *   - song_info：谱面等级与难度，格式为 "<level>_<difficulty>"，例如 "13.0_mas"。
 */
struct TicketItem {
  1: required string title,
  2: required string keyword,
  3: required string creator,
  4: required string image,
  5: required string cover_info,
  6: required string genre_info,
  7: required string song_info,
}

/*
 * 组件状态快照。
 *
 * 字段：
 *   - name：组件名。当前固定支持：
 *       - "server"：HTTP 服务组件。
 *       - "danmu"：Bilibili 弹幕监听组件。
 *       - "blivedm"：Bilibili 普通直播弹幕监听组件，基于 blivedm-go。
 *       - "songs"：歌曲库组件。
 *       - "tickets"：歌单/点歌队列组件。
 *       - "messages"：消息组件。
 *   - status：组件当前状态字符串，见 ComponentStatus 注释。
 *   - error：最近一次错误信息；无错误时为空字符串。
 *   - params：组件当前系统参数。所有参数默认空字符串，PATCH 更新后持久化到 runtime/component.params.json。
 *   - events：组件内部重要事件，最多保留最近 100 条，超过后滚动删除最老事件。
 *
 * 当前组件参数：
 *   - server.danmu_source：弹幕实现选择。空字符串或 "danmu" 使用开放平台 danmu；"blivedm" 使用 blivedm。
 *   - danmu.app_id：Bilibili 直播开放平台项目 ID。必须是整数字符串。
 *   - danmu.anchor_code：主播身份码。
 *   - danmu.bilibili_ak_id：本地签名 AccessKey ID。可为空。
 *   - danmu.bilibili_ak_secret：本地签名 AccessKey Secret。可为空。
 *   - blivedm.room_id：Bilibili 直播间号。必须是大于 0 的整数字符串。
 *   - blivedm.cookie：已登录 Bilibili 账号 Cookie。可为空；为空时可能受 B 站反爬和限流影响。
 *   - songs.song_package_path：舞萌歌曲数据包目录，目录内需要包含 Music.xml。
 *   - songs.alias_file_path：alias JSON 文件路径。可为空。
 *   - songs.cache_path：歌曲库临时缓存 JSON 路径。可为空；非空时 songs 组件启动优先读取缓存，不再扫描 song_package_path。
 *   - server：当前无参数。
 *   - tickets：当前无参数。
 */
struct ComponentSnapshot {
  1: required ComponentName name,
  2: required ComponentStatusText status,
  3: required string error,
  4: required map<string, string> params,
  5: required list<ComponentEvent> events,
}

/*
 * GET /api/tickets 成功响应的 data 字段。
 *
 * 字段：
 *   - tickets：当前展示用点歌列表。后端会补足空槽位，因此通常为固定长度 12。
 */
struct GetTicketsData {
  1: required list<TicketItem> tickets,
}

/*
 * GET /api/tickets 成功响应。
 *
 * JSON 形状：
 *   {"data": {"tickets": [...]}}
 */
struct GetTicketsResponse {
  1: required GetTicketsData data,
}

/*
 * GET /api/messages 成功响应的 data 字段。
 *
 * 字段：
 *   - messages：最近的消息，最多保留 20 条。
 */
struct GetMessagesData {
  1: required list<Message> messages,
}

/*
 * GET /api/messages 成功响应。
 *
 * JSON 形状：
 *   {"data": {"messages": [...]}}
 */
struct GetMessagesResponse {
  1: required GetMessagesData data,
}

/*
 * GET /api/event/:caller/:event/:content 成功响应。
 *
 * 字段：
 *   - data：操作结果消息，例如 "成功！"、"关闭成功"、"切换成功"。
 */
struct EventResponse {
  1: required string data,
}

/*
 * GET /api/sysinfo 和 GET /api/components 成功响应。
 *
 * 字段：
 *   - data：所有已注册组件的状态快照数组。
 */
struct ComponentsResponse {
  1: required list<ComponentSnapshot> data,
}

/*
 * GET /api/component-event-types 成功响应的 data 字段。
 *
 * 字段：
 *   - types：当前后端会记录的组件事件类型列表。
 */
struct ComponentEventTypesData {
  1: required list<ComponentEventTypeInfo> types,
}

/*
 * GET /api/component-event-types 成功响应。
 *
 * JSON 形状：
 *   {"data": {"types": [...]}}
 */
struct ComponentEventTypesResponse {
  1: required ComponentEventTypesData data,
}

/*
 * PATCH /api/components/:name/params 请求体。
 *
 * 字段：
 *   - params：要更新的参数键值。只允许更新目标组件声明过的参数；
 *     传未知参数会返回 400。
 *
 * 兼容说明：
 *   当前后端也兼容直接传 map<string,string> 作为请求体，
 *   例如 {"app_id":"123"}；推荐前端使用 {"params": {...}}。
 *
 * 重要行为：
 *   PATCH 只保存参数到 runtime/component.params.json，不会自动重启组件。
 *   如需生效，请随后调用 POST /api/components/:name/restart。
 *   对 songs 组件，cache_path 优先级高于 song_package_path；cache_path 为空时才会扫描 package 并 dump 默认缓存。
 */
struct UpdateComponentParamsRequest {
  1: required map<string, string> params,
}

/*
 * PATCH /api/components/:name/params 成功响应。
 *
 * 字段：
 *   - data：更新参数后的组件状态快照。status 不会因为 PATCH 自动变更为 restarting/running。
 */
struct UpdateComponentParamsResponse {
  1: required ComponentSnapshot data,
}

/*
 * POST /api/components/:name/restart 成功响应。
 *
 * 字段：
 *   - data：重启后的组件状态快照。
 *
 * 重启语义：
 *   - name="danmu"：停止旧弹幕监听并重新读取当前 danmu 参数启动。
 *   - name="blivedm"：停止旧 blivedm 监听并重新读取当前 blivedm 参数启动。
 *   - 重启 danmu 或 blivedm 前，后端会先停止另一个弹幕组件，避免双监听。
 *   - name="songs"：重新加载歌曲库和 alias。
 *   - name="tickets"：重新加载 tickets checkpoint，并绑定当前 songs 存储。
 *   - name="messages"：重新加载 messages checkpoint。
 *   - name="server"：当前不支持通过 HTTP 重启自身；会返回 400。
 */
struct RestartComponentResponse {
  1: required ComponentSnapshot data,
}

/*
 * POST /api/components/:name/restart 失败响应。
 *
 * 字段：
 *   - msg：错误信息。
 *   - data：失败时的组件状态快照；例如 server restart 不支持时会返回 server 组件快照。
 */
struct RestartComponentError {
  1: required string msg,
  2: required ComponentSnapshot data,
}

/*
 * POST /api/components/:name/stop 成功响应。
 *
 * 字段：
 *   - data：停止后的组件状态快照。
 *
 * 停止语义：
 *   - 只停止目标组件，不联动停止 danmu/blivedm peer。
 *   - 不修改组件参数，不清空 checkpoint。
 *   - name="server"：当前不支持通过 HTTP 停止自身；会返回 400。
 */
struct StopComponentResponse {
  1: required ComponentSnapshot data,
}

/*
 * POST /api/components/:name/stop 失败响应。
 *
 * 字段：
 *   - msg：错误信息。
 *   - data：失败时的组件状态快照；例如 server stop 不支持时会返回 server 组件快照。
 */
struct StopComponentError {
  1: required string msg,
  2: required ComponentSnapshot data,
}

/*
 * POST /sign 请求体。
 *
 * 字段：
 *   - req_json：待签名的 Bilibili 开放平台请求 JSON 字符串。
 *   - anchor_code：主播身份码。当前服务端只透传到请求结构；实际本地签名只依赖 AK/SK。
 */
struct RemoteSignRequest {
  1: required string req_json,
  2: required string anchor_code,
}

/*
 * Bilibili 开放平台签名头。
 *
 * 字段：
 *   - content_type：HTTP Content-Type，当前为 "application/json"。
 *   - content_accept_type：HTTP Accept，当前为 "application/json"。
 *   - timestamp：签名时间戳，Unix 秒级字符串。
 *   - signature_method：签名算法，当前为 "HMAC-SHA256"。
 *   - signature_version：签名版本，当前为 "1.0"。
 *   - authorization：最终 HMAC-SHA256 签名值。
 *   - nonce：随机串，当前使用 Unix 纳秒时间戳字符串。
 *   - access_key_id：Bilibili AccessKey ID。
 *   - content_md5：req_json 的 MD5 值。
 */
struct CommonHeader {
  1: required string content_type,
  2: required string content_accept_type,
  3: required string timestamp,
  4: required string signature_method,
  5: required string signature_version,
  6: required string authorization,
  7: required string nonce,
  8: required string access_key_id,
  9: required string content_md5,
}

/*
 * POST /sign 成功响应。
 *
 * 字段：
 *   - signed：Bilibili 开放平台请求所需签名头字段。
 */
struct RemoteSignResponse {
  1: required CommonHeader signed,
}

/*
 * 本地 Wolfy HTTP API。
 *
 * 注意：这些方法名是对 HTTP API 的 Thrift 化描述，不代表当前后端真实暴露 Thrift RPC。
 */
service WolfyLocalHTTPAPI {
  /*
   * GET /static/*filepath
   *
   * 作用：
   *   获取内置前端静态资源，例如 /static/、/static/js/...、/static/css/...。
   *
   * 参数：
   *   - filepath：静态资源相对路径。访问 /static/ 时由静态文件服务返回入口页面。
   *
   * 返回值：
   *   - 二进制静态文件内容，由 Gin static file handler 处理；不是 JSON。
   */
  binary getStaticAsset(1: string filepath);

  /*
   * GET /api/tickets
   *
   * 作用：
   *   获取当前点歌展示列表，用于舞台页和管理页展示封面、曲名、点歌人、谱面等信息。
   *
   * 参数：
   *   无。
   *
   * 返回值：
   *   - data.tickets：点歌展示项列表。每项字段见 TicketItem。
   *
   * 错误：
   *   - 503 {"msg": "..."}：tickets 组件未运行或未注册。
   */
  GetTicketsResponse getTickets();

  /*
   * GET /api/messages
   *
   * 作用：
   *   获取最近的操作反馈消息，例如点歌成功、切换成功、权限错误、曲库为空等。
   *
   * 参数：
   *   无。
   *
   * 返回值：
   *   - data.messages：最近的消息列表。每项字段见 Message。
   *
   * 错误：
   *   - 503 {"msg": "..."}：messages 组件未运行或未注册。
   */
  GetMessagesResponse getMessages();

  /*
   * GET /api/event/:caller/:event/:content
   *
   * 作用：
   *   前端触发点歌队列操作。该接口会被转换为内部 Task，再交给 tickets 组件执行。
   *
   * 路径参数：
   *   - caller：操作者名称。当前前端通常传 "主播"；
   *     权限判断中 "主播" 是 super admin，可操作任意点歌。
   *   - event：前端事件字符串，见 FrontendEventType 注释。
   *   - content：
   *       - event="pick" 或 "click_creator" 时：歌曲关键词。
   *       - event 为 click_cover_info/click_genre_info/click_song_info 时：点歌列表 0-based index 字符串。
   *
   * 返回值：
   *   - data：操作结果文案。
   *
   * 错误：
   *   - 400 {"msg": "..."}：操作失败，例如编号错误、权限不足、歌曲未匹配、tickets 未运行。
   */
  EventResponse emitFrontendEvent(
    1: string caller,
    2: FrontendEvent event,
    3: string content,
  );

  /*
   * GET /api/sysinfo
   *
   * 作用：
   *   获取所有组件的状态、错误信息和当前系统参数。
   *
   * 参数：
   *   无。
   *
   * 返回值：
   *   - data：组件状态快照数组。每项字段见 ComponentSnapshot。
   */
  ComponentsResponse getSysInfo();

  /*
   * GET /api/components
   *
   * 作用：
   *   GET /api/sysinfo 的别名。用于外部管理工具按组件语义读取状态。
   *
   * 参数：
   *   无。
   *
   * 返回值：
   *   - data：组件状态快照数组。每项字段见 ComponentSnapshot。
   */
  ComponentsResponse getComponents();

  /*
   * GET /api/component-event-types
   *
   * 作用：
   *   获取所有组件事件类型及说明，方便前端展示事件列表和筛选项。
   *
   * 参数：
   *   无。
   *
   * 返回值：
   *   - data.types：事件类型说明数组。
   */
  ComponentEventTypesResponse getComponentEventTypes();

  /*
   * PATCH /api/components/:name/params
   *
   * 作用：
   *   修改某个组件的系统参数并持久化到 runtime/component.params.json。
   *   该接口不会自动重启组件。
   *
   * 路径参数：
   *   - name：组件名，当前支持 "server"、"danmu"、"blivedm"、"songs"、"tickets"。
   *
   * 请求体：
   *   - params：要更新的参数键值。字段见 UpdateComponentParamsRequest。
   *
   * 返回值：
   *   - data：更新参数后的组件状态快照。
   *
   * 错误：
   *   - 400 {"msg": "..."}：组件不存在、JSON 格式错误、参数名不属于该组件、文件写入失败等。
   */
  UpdateComponentParamsResponse updateComponentParams(
    1: ComponentName name,
    2: UpdateComponentParamsRequest request,
  );

  /*
   * POST /api/components/:name/restart
   *
   * 作用：
   *   独立重启某个组件，使已经保存的参数生效。
   *
   * 路径参数：
   *   - name：组件名，当前支持 "server"、"danmu"、"blivedm"、"songs"、"tickets"。
   *
   * 请求体：
   *   无。
   *
   * 返回值：
   *   - data：重启后的组件状态快照。
   *
   * 错误：
   *   - 400 {"msg": "...", "data": {...}}：组件不存在、server 不支持 HTTP 自重启、启动参数错误等。
   */
  RestartComponentResponse restartComponent(1: ComponentName name);

  /*
   * POST /api/components/:name/stop
   *
   * 作用：
   *   独立停止某个组件。
   *
   * 路径参数：
   *   - name：组件名，当前支持 "server"、"danmu"、"blivedm"、"songs"、"tickets"、"messages"。
   *
   * 请求体：
   *   无。
   *
   * 返回值：
   *   - data：停止后的组件状态快照。
   *
   * 错误：
   *   - 400 {"msg": "...", "data": {...}}：组件不存在、server 不支持 HTTP 停止、停止过程出错等。
   */
  StopComponentResponse stopComponent(1: ComponentName name);
}

/*
 * 远程签名 HTTP API。
 *
 * 注意：该服务只在 remote build tag 下启动，和本地 41377 服务不是同一个 HTTP server。
 */
service WolfyRemoteSignHTTPAPI {
  /*
   * POST /sign
   *
   * 作用：
   *   根据远程签名服务本地配置的 BILIBILI_AK_ID/BILIBILI_AK_SECRET，
   *   为传入的 Bilibili 开放平台请求 JSON 生成签名头。
   *
   * 请求体：
   *   - req_json：待签名请求 JSON 字符串。
   *   - anchor_code：主播身份码。
   *
   * 返回值：
   *   - signed：签名头对象。字段见 CommonHeader。
   *
   * 错误：
   *   - 400 {"message": "..."}：请求 JSON 绑定失败。
   *   - 500 {"message": "..."}：签名失败。
   */
  RemoteSignResponse sign(1: RemoteSignRequest request);
}
