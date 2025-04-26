package service

import (
	"encoding/json"
	fuzz "github.com/paul-mannino/go-fuzzywuzzy"
	"os"
	"sort"
)

type MaimaiStorage struct {
	filePath string
	records  map[string]*MaimaiRecord
}

type item struct {
	score int
	id    string
}

func NewMaimaiStorage(filePath string) *MaimaiStorage {
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	res := &MaimaiStorage{
		filePath: filePath,
	}

	err = json.Unmarshal(file, &res.records)
	if err != nil {
		panic(err)
	}

	return res
}

func (s *MaimaiStorage) PickOne(keyword string, rank int) *MaimaiRecord {
	rankList := s.rankRecord(keyword)
	return s.records[rankList[rank%len(rankList)].id]
}

func (s *MaimaiStorage) rankRecord(keyword string) []*item {

	var result = make([]*item, 0, len(s.records))
	for id, record := range s.records {
		highScore := -1
		for _, alias := range record.Alias {
			score := fuzz.Ratio(alias, keyword)
			if alias == keyword {
				score++
			}
			if score > highScore {
				highScore = score
			}
		}
		result = append(result, &item{
			id:    id,
			score: highScore,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].score == result[j].score {
			return result[i].id < result[j].id
		} else {
			return result[i].score > result[j].score
		}
	})

	return result
}
