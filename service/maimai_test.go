package service

import (
	"fmt"
	"testing"
)

func TestTicketMaster(t *testing.T) {
	localTicketsCheckPointPath := "../runtime/tickets.checkpoint.json"

	localSongDBPath := "../static/maimai/songs.json"
	tm := NewMaimaiTicketMaster(
		localSongDBPath,
		localTicketsCheckPointPath, 12, 3)
	testCases := map[string]struct {
		title     string
		diff      string
		trackType string
	}{
		"标准夜骑":   {"ナイト・オブ・ナイツ", "14", "std"},
		"夜骑标准":   {"ナイト・オブ・ナイツ", "14", "std"},
		"白夜骑标准":  {"ナイト・オブ・ナイツ", "14", "std"},
		"标准紫夜骑":  {"ナイト・オブ・ナイツ", "13", "std"},
		"紫标准夜骑":  {"ナイト・オブ・ナイツ", "13", "std"},
		"紫std夜骑": {"ナイト・オブ・ナイツ", "13", "std"},
		"紫DX夜骑":  {"ナイト・オブ・ナイツ", "13", "dx"},
		"紫dx夜骑":  {"ナイト・オブ・ナイツ", "13", "dx"},
		"DX紫夜骑":  {"ナイト・オブ・ナイツ", "13", "dx"},
		"标准白夜骑":  {"ナイト・オブ・ナイツ", "14", "std"},
		"白雪":     {"白ゆき", "13", "std"},
		"白花一轮":   {"花と、雪と、ドラムンベース。", "13+", "std"},
		"紫花一轮":   {"花と、雪と、ドラムンベース。", "14", "std"},
		"红潘":     {"PANDORA PARADOXXX", "13+", "std"},
		"CF":     {"Calamity Fortune", "13+", "dx"},
		"DXCF":   {"Calamity Fortune", "13+", "dx"},
		"标准CF":   {"Calamity Fortune", "14", "std"},
	}
	for keyword, expected := range testCases {
		fmt.Printf("Start smart pick %s\n", keyword)
		pick, err := tm.SmartPick(keyword, 0)
		if err != nil {
			return
		}
		if pick.Title != expected.title {
			t.Errorf("%s %s unexpected => %s", keyword, pick.Title, expected.title)
		}
		if pick.GetTrackDifficulty() != expected.diff {
			t.Errorf("%s %s unexpected => %s", keyword, pick.GetTrackDifficulty(), expected.diff)
		}
		if pick.GetTrackType() != expected.trackType {
			t.Errorf("%s %s unexpected => %s", keyword, pick.GetTrackType(), expected.trackType)
		}
	}

}
