package bilibili

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
	"wolfy/model"
)

const (
	MaxBodySize     = int32(1 << 11)
	CmdSize         = 4
	PackSize        = 4
	HeaderSize      = 2
	VerSize         = 2
	OperationSize   = 4
	SeqIdSize       = 4
	HeartbeatSize   = 4
	RawHeaderSize   = PackSize + HeaderSize + VerSize + OperationSize + SeqIdSize
	MaxPackSize     = MaxBodySize + int32(RawHeaderSize)
	PackOffset      = 0
	HeaderOffset    = PackOffset + PackSize
	VerOffset       = HeaderOffset + HeaderSize
	OperationOffset = VerOffset + VerSize
	SeqIdOffset     = OperationOffset + OperationSize
	HeartbeatOffset = SeqIdOffset + SeqIdSize
)

const (
	OP_HEARTBEAT       = int32(2)
	OP_HEARTBEAT_REPLY = int32(3)
	OP_SEND_SMS_REPLY  = int32(5)
	OP_AUTH            = int32(7)
	OP_AUTH_REPLY      = int32(8)
)

type WebsocketClient struct {
	conn       *websocket.Conn
	msgBuf     chan *Proto
	sequenceId int32
	dispatcher map[int32]protoLogic
	authed     bool
	taskChan   chan *model.Task
}

type protoLogic func(p *Proto) (err error)

type Proto struct {
	PacketLength int32
	HeaderLength int16
	Version      int16
	Operation    int32
	SequenceId   int32
	Body         []byte
	BodyMuti     [][]byte
}

type AuthRespParam struct {
	Code int64 `json:"code,omitempty"`
}

// StartWebsocket 启动长连
func StartWebsocket(ctx context.Context, wsAddr, authBody string, taskChan chan *model.Task) (err error) {
	// 建立连接
	conn, _, err := websocket.DefaultDialer.Dial(wsAddr, nil)
	if err != nil {
		return err
	}
	wc := &WebsocketClient{
		conn:       conn,
		msgBuf:     make(chan *Proto, 1024),
		dispatcher: make(map[int32]protoLogic),
		taskChan:   taskChan,
	}

	// 注册分发处理函数
	wc.dispatcher[OP_AUTH_REPLY] = wc.authResp
	wc.dispatcher[OP_HEARTBEAT_REPLY] = wc.heartBeatResp
	wc.dispatcher[OP_SEND_SMS_REPLY] = wc.msgResp

	// 发送鉴权信息
	err = wc.sendAuth(authBody)
	if err != nil {
		return
	}

	// 读取信息
	go wc.ReadMsg(ctx)

	// 处理信息
	go wc.DoEvent(ctx)

	return
}

// ReadMsg 读取长连信息
func (wc *WebsocketClient) ReadMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("[WebsocketClient | ReadMsg] ctx done")
			err := wc.conn.Close()
			if err != nil {
				log.Fatalf("[WebsocketClient | ReadMsg] ctx done: failed to close connection %v", err)
			}
			return
		default:
			retProto := &Proto{}
			_, buf, err := wc.conn.ReadMessage()
			if err != nil {
				log.Println("[WebsocketClient | ReadMsg] err:", err.Error())
				continue
			}
			retProto.PacketLength = int32(binary.BigEndian.Uint32(buf[PackOffset:HeaderOffset]))
			retProto.HeaderLength = int16(binary.BigEndian.Uint16(buf[HeaderOffset:VerOffset]))
			retProto.Version = int16(binary.BigEndian.Uint16(buf[VerOffset:OperationOffset]))
			retProto.Operation = int32(binary.BigEndian.Uint32(buf[OperationOffset:SeqIdOffset]))
			retProto.SequenceId = int32(binary.BigEndian.Uint32(buf[SeqIdOffset:]))
			if retProto.PacketLength < 0 || retProto.PacketLength > MaxPackSize {
				continue
			}
			if retProto.HeaderLength != RawHeaderSize {
				continue
			}
			if bodyLen := int(retProto.PacketLength - int32(retProto.HeaderLength)); bodyLen > 0 {
				retProto.Body = buf[retProto.HeaderLength:retProto.PacketLength]
			} else {
				continue
			}
			retProto.BodyMuti = [][]byte{retProto.Body}
			if len(retProto.BodyMuti) > 0 {
				retProto.Body = retProto.BodyMuti[0]
			}
			wc.msgBuf <- retProto
		}
	}
}

// DoEvent 处理信息
func (wc *WebsocketClient) DoEvent(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			log.Println("[WebsocketClient | DoEvent] ctx done")
			return
		case p := <-wc.msgBuf:
			if p == nil {
				continue
			}
			if wc.dispatcher[p.Operation] == nil {
				continue
			}
			err := wc.dispatcher[p.Operation](p)
			if err != nil {
				continue
			}
		case <-ticker.C:
			wc.sendHeartBeat()
		}
	}
}

// sendAuth 发送鉴权
func (wc *WebsocketClient) sendAuth(authBody string) (err error) {
	p := &Proto{
		Operation: OP_AUTH,
		Body:      []byte(authBody),
	}
	return wc.sendMsg(p)
}

// sendHeartBeat 发送心跳
func (wc *WebsocketClient) sendHeartBeat() {
	if !wc.authed {
		return
	}
	msg := &Proto{}
	msg.Operation = OP_HEARTBEAT
	msg.SequenceId = wc.sequenceId
	wc.sequenceId++
	err := wc.sendMsg(msg)
	if err != nil {
		return
	}
	log.Println("[WebsocketClient | sendHeartBeat] seq:", msg.SequenceId)
}

// sendMsg 发送信息
func (wc *WebsocketClient) sendMsg(msg *Proto) (err error) {
	dataBuff := &bytes.Buffer{}
	packLen := int32(RawHeaderSize + len(msg.Body))
	msg.HeaderLength = RawHeaderSize
	binary.Write(dataBuff, binary.BigEndian, packLen)
	binary.Write(dataBuff, binary.BigEndian, int16(RawHeaderSize))
	binary.Write(dataBuff, binary.BigEndian, msg.Version)
	binary.Write(dataBuff, binary.BigEndian, msg.Operation)
	binary.Write(dataBuff, binary.BigEndian, msg.SequenceId)
	binary.Write(dataBuff, binary.BigEndian, msg.Body)
	err = wc.conn.WriteMessage(websocket.BinaryMessage, dataBuff.Bytes())
	if err != nil {
		log.Println("[WebsocketClient | sendMsg] send msg err:", msg)
		return
	}
	return
}

// authResp 鉴权处理函数
func (wc *WebsocketClient) authResp(msg *Proto) (err error) {
	resp := &AuthRespParam{}
	err = json.Unmarshal(msg.Body, resp)
	if err != nil {
		return
	}
	if resp.Code != 0 {
		return
	}
	wc.authed = true
	log.Println("[WebsocketClient | authResp] auth success")
	return
}

// heartBeatResp  心跳结果
func (wc *WebsocketClient) heartBeatResp(msg *Proto) (err error) {
	log.Println("[WebsocketClient | heartBeatResp] get HeartBeat resp", msg.Body)
	return
}

// msgResp 可以这里做回调
func (wc *WebsocketClient) msgResp(msg *Proto) (err error) {
	for index, cmd := range msg.BodyMuti {
		log.Printf("[WebsocketClient | msgResp] recv MsgResp "+
			"index:%d ver:%d cmd:%s", index, msg.Version, string(cmd))
		var r RespMessage
		err = json.Unmarshal(cmd, &r)
		if err != nil {
			continue
		}
		if wc.taskChan == nil {
			continue
		}
		var task *model.Task
		if r.Cmd == OpenPlatformDanmuCmd {
			task = parseDanmu(r.Data.Uname, r.Data.Msg)
		}
		wc.taskChan <- task
	}
	return
}
