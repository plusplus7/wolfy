package bilibili

import (
	"encoding/binary"
	"testing"
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
