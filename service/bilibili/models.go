package bilibili

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

const (
	OpenPlatformHttpHost = "https://live-open.biliapi.com" //开放平台 (线上环境)
	OpenPlatformDanmuCmd = "LIVE_OPEN_PLATFORM_DM"
)

type StartAppRequest struct {
	// 主播身份码
	Code string `json:"code"`
	// 项目id
	AppId int64 `json:"app_id"`
}

type StartAppRespData struct {
	// 场次信息
	GameInfo GameInfo `json:"game_info"`
	// 长连信息
	WebsocketInfo WebSocketInfo `json:"websocket_info"`
	// 主播信息
	AnchorInfo AnchorInfo `json:"anchor_info"`
}

type GameInfo struct {
	GameId string `json:"game_id"`
}

type WebSocketInfo struct {
	//  长连使用的请求json体 第三方无需关注内容,建立长连时使用即可
	AuthBody string `json:"auth_body"`
	//  wss 长连地址
	WssLink []string `json:"wss_link"`
}

type AnchorInfo struct {
	//主播房间号
	RoomId int64 `json:"room_id"`
	//主播昵称
	Uname string `json:"uname"`
	//主播头像
	Uface string `json:"uface"`
	//主播uid
	Uid int64 `json:"uid"`
	//主播open_id
	OpenId string `json:"open_id"`
}

type EndAppRequest struct {
	// 场次id
	GameId string `json:"game_id"`
	// 项目id
	AppId int64 `json:"app_id"`
}

type AppHeartbeatReq struct {
	// 主播身份码
	GameId string `json:"game_id"`
}

type RespMessage struct {
	Data struct {
		EmojiImgURL            string `json:"emoji_img_url"`
		FansMedalLevel         int    `json:"fans_medal_level"`
		FansMedalName          string `json:"fans_medal_name"`
		FansMedalWearingStatus bool   `json:"fans_medal_wearing_status"`
		GuardLevel             int    `json:"guard_level"`
		Msg                    string `json:"msg"`
		Timestamp              int    `json:"timestamp"`
		UID                    int    `json:"uid"`
		Uname                  string `json:"uname"`
		Uface                  string `json:"uface"`
		DmType                 int    `json:"dm_type"`
		Open                   string `json:"open_"`
		IsAdmin                int    `json:"is_admin"`
		GloryLevel             int    `json:"glory_level"`
		ReplyOpenID            string `json:"reply_open_id"`
		ReplyUname             string `json:"reply_uname"`
		MsgID                  string `json:"msg_id"`
		RoomID                 int    `json:"room_id"`
	} `json:"data"`
	Cmd string `json:"cmd"`
}

const (
	AcceptHeader              = "Accept"
	ContentTypeHeader         = "Content-Type"
	AuthorizationHeader       = "Authorization"
	JsonType                  = "application/json"
	BiliVersion               = "1.0"
	HmacSha256                = "HMAC-SHA256"
	BiliTimestampHeader       = "x-bili-timestamp"
	BiliSignatureMethodHeader = "x-bili-signature-method"
	BiliSignatureNonceHeader  = "x-bili-signature-nonce"
	BiliAccessKeyIdHeader     = "x-bili-accesskeyid"
	BiliSignVersionHeader     = "x-bili-signature-version"
	BiliContentMD5Header      = "x-bili-content-md5"
)

type CommonHeader struct {
	ContentType       string `json:"content_type"`
	ContentAcceptType string `json:"content_accept_type"`
	Timestamp         string `json:"timestamp"`
	SignatureMethod   string `json:"signature_method"`
	SignatureVersion  string `json:"signature_version"`
	Authorization     string `json:"authorization"`
	Nonce             string `json:"nonce"`
	AccessKeyId       string `json:"access_key_id"`
	ContentMD5        string `json:"content_md5"`
}

func (h *CommonHeader) ToHTTPHeader() http.Header {
	return http.Header{
		BiliTimestampHeader:       {h.Timestamp},
		BiliSignatureMethodHeader: {h.SignatureMethod},
		BiliSignatureNonceHeader:  {h.Nonce},
		BiliAccessKeyIdHeader:     {h.AccessKeyId},
		BiliSignVersionHeader:     {h.SignatureVersion},
		BiliContentMD5Header:      {h.ContentMD5},
		AuthorizationHeader:       {h.Authorization},
		ContentTypeHeader:         {h.ContentType},
		AcceptHeader:              {h.ContentAcceptType},
	}
}

// ToMap 所有字段转map<string, string>
func (h *CommonHeader) ToMap() map[string]string {
	return map[string]string{
		BiliTimestampHeader:       h.Timestamp,
		BiliSignatureMethodHeader: h.SignatureMethod,
		BiliSignatureNonceHeader:  h.Nonce,
		BiliAccessKeyIdHeader:     h.AccessKeyId,
		BiliSignVersionHeader:     h.SignatureVersion,
		BiliContentMD5Header:      h.ContentMD5,
		AuthorizationHeader:       h.Authorization,
		ContentTypeHeader:         h.ContentType,
		AcceptHeader:              h.ContentAcceptType,
	}
}

// ToSortMap 参与加密的字段转map<string, string>
func (h *CommonHeader) ToSortMap() map[string]string {
	return map[string]string{
		BiliTimestampHeader:       h.Timestamp,
		BiliSignatureMethodHeader: h.SignatureMethod,
		BiliSignatureNonceHeader:  h.Nonce,
		BiliAccessKeyIdHeader:     h.AccessKeyId,
		BiliSignVersionHeader:     h.SignatureVersion,
		BiliContentMD5Header:      h.ContentMD5,
	}
}

// ToSortedString 生成需要加密的文本
func (h *CommonHeader) ToSortedString() (sign string) {
	hMap := h.ToSortMap()
	var hSil []string
	for k := range hMap {
		hSil = append(hSil, k)
	}
	sort.Strings(hSil)
	for _, v := range hSil {
		sign += v + ":" + hMap[v] + "\n"
	}
	sign = strings.TrimRight(sign, "\n")
	return
}

type BaseResp struct {
	Code      int64           `json:"code"`
	Message   string          `json:"message"`
	RequestId string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
}
