package service

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	fuzz "github.com/paul-mannino/go-fuzzywuzzy"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type MaimaiStorage struct {
	filePath string
	records  map[int]*MaimaiRecord
	aliases  map[int][]string
}

type item struct {
	score int
	id    int
}

func coverPath(id int) string {
	return "https://assets2.lxns.net/maimai/jacket/" + strconv.Itoa(id) + ".png"
}
func parseSongInfoFromXML(path string) (*MaimaiRecord, error) {
	var music MusicData
	xmlFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	// Read file content
	data, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, err
	}
	if err = xml.Unmarshal(data, &music); err != nil {
		return nil, err
	}

	musicID := music.Name.ID
	coverUrl := ""
	noteType := ""
	if musicID >= 10000 && musicID < 100000 {
		noteType = "dx"
		coverUrl = coverPath(musicID - 10000)
	} else if musicID >= 100000 {
		noteType = "宴"
		coverUrl = coverPath(musicID - 100000)
	} else {
		noteType = "std"
		coverUrl = coverPath(musicID)
	}
	var levels []MaimaiLevel
	difficulties := []string{"bas", "adv", "exp", "mas", "remas"}
	for i, note := range music.NotesData.Notes {
		if note.Level != 0 {
			levels = append(levels, MaimaiLevel{
				Type:       noteType,
				Difficulty: difficulties[i],
				Level:      strconv.Itoa(note.Level) + "." + strconv.Itoa(note.LevelDecimal),
			})
		}
	}
	result := &MaimaiRecord{
		ID:        musicID,
		Title:     music.Name.Str,
		ImagePath: coverUrl,
		Levels:    levels,
		Category:  music.GenreName.Str,
	}

	if genre, ok := genreMapping[music.GenreName.Str]; ok {
		result.Category = genre
	} else {
		result.Category = "舞萌"
		log.Printf("Couldn't find genre mapping for %s %s", path, music.GenreName.Str)
	}
	return result, nil
}

var genreMapping = map[string]string{
	"ゲームバラエティ":     "其他游戏",
	"maimai":               "舞萌",
	"POPSアニメ":           "流行&动漫",
	"niconicoボーカロイド": "niconico＆VOCALOID™",
	"東方Project":          "东方Project",
	"オンゲキCHUNITHM":     "音击/中二节奏",
	"宴会場":               "宴会場",
}

func collectAlias(aliasPath string) (*Aliases, error) {
	tryFetch, err := fetchAliasList()
	if err != nil {
		log.Printf("Failed to fetch aliases %v\n", err)
	} else {
		return tryFetch, nil
	}

	var aliases Aliases
	if aliasPath != "" {
		aliasFile, err := os.Open(aliasPath)
		if err != nil {
			return nil, err
		}
		defer aliasFile.Close()

		aliasContent, err := io.ReadAll(aliasFile)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(aliasContent, &aliases)
		if err != nil {
			return nil, err
		}
	}
	return &aliases, nil
}

func collectSongInfoFromPackage(path string, aliasPath string) (*MaimaiStorage, error) {

	aliases, err := collectAlias(aliasPath)
	if err != nil {
		return nil, err
	}
	storage := &MaimaiStorage{
		filePath: path,
		records:  map[int]*MaimaiRecord{},
		aliases:  map[int][]string{},
	}

	err = filepath.WalkDir(path,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Fatalf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
				return err
			}
			if d.IsDir() == false && d.Name() == "Music.xml" {
				fromXML, err := parseSongInfoFromXML(path)
				if err != nil {
					log.Printf("error parsing song info from %s: %v\n", path, err)
					return err
				}
				targetID := fromXML.ID
				if fromXML.ID >= 10000 && fromXML.ID < 20000 {
					targetID = fromXML.ID - 10000
				} else if fromXML.ID > 100000 {
					targetID = fromXML.ID - 100000
				} else {
					targetID = fromXML.ID
				}
				storage.records[targetID] = fromXML
				if _, ok := storage.aliases[targetID]; !ok {
					storage.aliases[targetID] = []string{fromXML.Title}
				}
			}
			//	log.Printf("visited file or dir: %q\n", path)
			return nil
		})

	for _, alias := range aliases.Alias {
		if _, ok := storage.records[alias.SongID]; ok {
			if _, ok2 := storage.aliases[alias.SongID]; !ok2 {
				storage.aliases[alias.SongID] = alias.Aliases
			} else {
				storage.aliases[alias.SongID] = append(storage.aliases[alias.SongID], alias.Aliases...)
			}
		}
	}
	log.Println("Songs found:", len(storage.aliases), "records:", len(storage.records))
	return storage, nil
}

func NewMaimaiStorage(filePath string, aliasPath string) *MaimaiStorage {
	fromPackage, err := collectSongInfoFromPackage(filePath, aliasPath)
	if err != nil {
		panic(err)
	}
	return fromPackage
}

func (s *MaimaiStorage) PickOne(keyword string, rank int) *MaimaiRecord {
	rankList := s.rankRecord(keyword)
	return s.records[rankList[rank%len(rankList)].id]
}

func (s *MaimaiStorage) rankRecord(keyword string) []*item {

	var result = make([]*item, 0, len(s.records))
	for id, aliases := range s.aliases {
		highScore := -1
		if id == 1681 {
			highScore = -1
		}
		for _, alias := range aliases {
			score := fuzz.UQRatio(alias, keyword)
			if strings.ToLower(alias) == strings.ToLower(keyword) {
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
	for i, r := range result {
		fmt.Println(r.score, r.id, s.records[r.id].Title)
		if i > 20 {
			break
		}
	}

	return result
}

func fetchAliasList() (*Aliases, error) {

	url := "https://maimai.lxns.net/api/v0/maimai/alias/list"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	aliases := &Aliases{}
	err = json.Unmarshal(body, aliases)
	if err != nil {
		return nil, err
	}
	return aliases, nil
}
