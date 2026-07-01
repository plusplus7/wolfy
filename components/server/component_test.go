package server

import "testing"

func TestServerComponentHasNoConfigurableParams(t *testing.T) {
	component := NewServerComponent()
	if got := component.ParamKeys(); len(got) != 0 {
		t.Fatalf("expected no server params, got %#v", got)
	}
}

func TestServerComponentRejectsUnknownParams(t *testing.T) {
	component := NewServerComponent()
	if err := component.ValidateParams(map[string]string{"danmu_source": "legacy"}); err == nil {
		t.Fatal("expected unknown parameter error for removed danmu_source")
	}
}
