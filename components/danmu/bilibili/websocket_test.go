package bilibili

import (
	"encoding/binary"
	"strings"
	"testing"
	"wolfy/components"
	"wolfy/model"
)

func packet(packetLength int32, headerLength int16, body []byte) []byte {
	buf := make([]byte, RawHeaderSize+len(body))
	binary.BigEndian.PutUint32(buf[PackOffset:HeaderOffset], uint32(packetLength))
	binary.BigEndian.PutUint16(buf[HeaderOffset:VerOffset], uint16(headerLength))
	binary.BigEndian.PutUint16(buf[VerOffset:OperationOffset], 0)
	binary.BigEndian.PutUint32(buf[OperationOffset:SeqIdOffset], uint32(OP_SEND_SMS_REPLY))
	binary.BigEndian.PutUint32(buf[SeqIdOffset:RawHeaderSize], 1)
	copy(buf[RawHeaderSize:], body)
	return buf
}

func TestParseProtoRejectsMalformedPackets(t *testing.T) {
	cases := [][]byte{
		{1, 2, 3},
		packet(int32(RawHeaderSize+100), RawHeaderSize, []byte(`{}`)),
		packet(int32(RawHeaderSize+2), RawHeaderSize+1, []byte(`{}`)),
		packet(int32(RawHeaderSize), RawHeaderSize, nil),
	}
	for _, tc := range cases {
		if proto, ok := parseProto(tc); ok || proto != nil {
			t.Fatalf("expected malformed packet to be rejected: %#v", proto)
		}
	}
}

func TestParseProtoAcceptsValidPacket(t *testing.T) {
	proto, ok := parseProto(packet(int32(RawHeaderSize+2), RawHeaderSize, []byte(`{}`)))
	if !ok {
		t.Fatal("expected valid packet")
	}
	if string(proto.Body) != `{}` {
		t.Fatalf("unexpected body %q", string(proto.Body))
	}
}

func TestMsgRespSkipsNonDanmuMessages(t *testing.T) {
	taskChan := make(chan *model.Task, 1)
	messageChan := make(chan string, 1)
	wc := &WebsocketClient{taskChan: taskChan, messageChan: messageChan}

	if err := wc.msgResp(&Proto{BodyMuti: [][]byte{
		[]byte(`{"cmd":"OTHER","data":{"uname":"alice","msg":"点歌 test"}}`),
	}}); err != nil {
		t.Fatal(err)
	}

	select {
	case task := <-taskChan:
		t.Fatalf("unexpected task %#v", task)
	default:
	}
	select {
	case message := <-messageChan:
		t.Fatalf("unexpected message %q", message)
	default:
	}
}

func TestMsgRespEmitsDanmuTasks(t *testing.T) {
	taskChan := make(chan *model.Task, 1)
	messageChan := make(chan string, 1)
	var recordedType components.ComponentEventType
	var recordedLocation string
	var recordedMessage string
	wc := &WebsocketClient{
		taskChan:    taskChan,
		messageChan: messageChan,
		recorder: func(eventType components.ComponentEventType, codeLocation, message string) {
			recordedType = eventType
			recordedLocation = codeLocation
			recordedMessage = message
		},
	}

	if err := wc.msgResp(&Proto{BodyMuti: [][]byte{
		[]byte(`{"cmd":"LIVE_OPEN_PLATFORM_DM","data":{"uname":"alice","msg":"点歌 test"}}`),
	}}); err != nil {
		t.Fatal(err)
	}

	task := <-taskChan
	if task.Command != model.CommandPick || task.Caller != "alice" || task.Content != "test" {
		t.Fatalf("unexpected task %#v", task)
	}
	message := <-messageChan
	if message != "inf alice 点歌 test" {
		t.Fatalf("unexpected message %q", message)
	}
	if recordedType != components.ComponentEventDanmuDanmuReceived {
		t.Fatalf("unexpected recorded type %q", recordedType)
	}
	if !strings.Contains(recordedLocation, "components/danmu/bilibili/websocket.go:") {
		t.Fatalf("unexpected recorded location %q", recordedLocation)
	}
	if recordedMessage != "caller=alice message=点歌 test" {
		t.Fatalf("unexpected recorded message %q", recordedMessage)
	}
}
