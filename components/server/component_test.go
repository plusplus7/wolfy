package server

import "testing"

func TestServerComponentDanmuSourceDefaultsToDanmu(t *testing.T) {
	component := NewServerComponent()
	if got := component.DanmuSource(); got != DanmuSourceDanmu {
		t.Fatalf("expected default source %q, got %q", DanmuSourceDanmu, got)
	}
}

func TestServerComponentValidatesDanmuSource(t *testing.T) {
	component := NewServerComponent()
	if err := component.ValidateParams(map[string]string{ParamDanmuSource: DanmuSourceBlivedm}); err != nil {
		t.Fatal(err)
	}
	if err := component.ValidateParams(map[string]string{ParamDanmuSource: "other"}); err == nil {
		t.Fatal("expected invalid source error")
	}
	if err := component.ValidateParams(map[string]string{"unknown": "value"}); err == nil {
		t.Fatal("expected unknown parameter error")
	}
}
