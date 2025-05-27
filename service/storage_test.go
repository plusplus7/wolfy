package service

import (
	"fmt"
	"testing"
)

func TestStorageRank(t *testing.T) {
	s := NewMaimaiStorage("../static/maimai/songs.json")
	records := s.RankRecord("白雪")
	for i, r := range records {
		fmt.Printf("%d,%s, %d\n", i, s.Records[r.id].Title, r.score)
	}
}
