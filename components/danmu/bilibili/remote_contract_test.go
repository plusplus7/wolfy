package bilibili

import (
	"reflect"
	"testing"
)

func TestRemoteDanmuContractJSONTags(t *testing.T) {
	assertJSONTag(t, reflect.TypeOf(StartGameRequest{}), "AppID", "app_id")
	assertJSONTag(t, reflect.TypeOf(StartGameRequest{}), "AnchorCode", "anchor_code")
	assertJSONTag(t, reflect.TypeOf(GameSession{}), "LastSeq", "last_seq")
	assertJSONTag(t, reflect.TypeOf(DanmuEvent{}), "AnchorCode", "anchor_code")
	assertJSONTag(t, reflect.TypeOf(DanmuEvent{}), "ReceivedAt", "received_at")
	assertJSONTag(t, reflect.TypeOf(DanmuEvent{}), "Task", "task,omitempty")
	assertJSONTag(t, reflect.TypeOf(PullDanmuResponse{}), "NextSeq", "next_seq")
	assertJSONTag(t, reflect.TypeOf(PullDanmuResponse{}), "HasMore", "has_more")
}

func assertJSONTag(t *testing.T, typ reflect.Type, fieldName string, want string) {
	t.Helper()
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		t.Fatalf("missing field %s on %s", fieldName, typ.Name())
	}
	if got := field.Tag.Get("json"); got != want {
		t.Fatalf("%s.%s json tag = %q, want %q", typ.Name(), fieldName, got, want)
	}
}
