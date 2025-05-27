package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"wolfy/model"
)

type MaimaiLevel struct {
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
	Level      string `json:"level"`
}

type MaimaiRecord struct {
	Rank      int           `json:"rank"`
	Title     string        `json:"title"`
	Alias     []string      `json:"alias"`
	ImagePath string        `json:"image"`
	Levels    []MaimaiLevel `json:"levels"`
	Category  string        `json:"category"`

	CurrentLevel int `json:"current_level"`
}

func (r *MaimaiRecord) GetTrackType() string {
	return r.Levels[r.CurrentLevel].Type
}

func (r *MaimaiRecord) GetTrackLevel() string {
	return r.Levels[r.CurrentLevel].Level
}

func (r *MaimaiRecord) GetTrackDifficulty() string {
	return r.Levels[r.CurrentLevel].Difficulty
}

func (r *MaimaiRecord) NextLevel() {
	r.CurrentLevel = (r.CurrentLevel + 1) % len(r.Levels)
}

type MaimaiTicket struct {
	Keyword string        `json:"keyword"`
	Creator string        `json:"creator"`
	Record  *MaimaiRecord `json:"record"`
	Rank    int           `json:"rank"`
}

func (m *MaimaiTicket) RotateLevel() {
	m.Record.NextLevel()
}

func (m *MaimaiTicket) GetKeyword() string {
	return m.Keyword
}

func (m *MaimaiTicket) GetCoverPath() string {
	return m.Record.ImagePath
}

func (m *MaimaiTicket) GetCoverInfo() string {
	return m.Record.GetTrackType()
}

func (m *MaimaiTicket) GetGenreInfo() string {
	return m.Record.Category
}

func (m *MaimaiTicket) GetSongInfo() string {
	return m.Record.GetTrackLevel() + "_" + m.Record.GetTrackDifficulty()
}

func (m *MaimaiTicket) GetTitle() string {
	return m.Record.Title
}

func (m *MaimaiTicket) GetCreator() string {
	return m.Creator
}

type MaimaiTicketMaster struct {
	lock    sync.RWMutex
	tickets []*MaimaiTicket

	maxTicketSize                  int
	maxAllowedTicketsForOneCreator int
	checkPointPath                 string
	storage                        *MaimaiStorage
}

func (t *MaimaiTicketMaster) ClearTickets(operator string) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.checkPermission(operator, -1) {
		return "", fmt.Errorf("管理员操作")
	}
	t.tickets = make([]*MaimaiTicket, 0)
	err := t.saveCheckPoint()
	if err != nil {
		log.Printf("failed to save ticket check point %v", err)
		return "", err
	}
	return "关闭成功", nil
}

const (
	superAdmin = "主播"
)

func (t *MaimaiTicketMaster) checkPermission(creator string, index int64) bool {
	if index >= int64(len(t.tickets)) {
		return false
	}
	if index == -1 {
		return creator == superAdmin
	}
	return t.tickets[index].Creator == creator || creator == superAdmin
}

func (t *MaimaiTicketMaster) FinishTicket(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if index >= int64(len(t.tickets)) {
		return "", fmt.Errorf("编号错误")
	}
	if index == -1 {
		for i, ticket := range t.tickets {
			if ticket.Creator == operator {
				index = int64(i)
				break
			}
		}
	}

	if !t.checkPermission(operator, index) {
		return "", fmt.Errorf("只能操作自己点的歌曲")
	}
	t.tickets = append(t.tickets[:index], t.tickets[index+1:]...)
	err := t.saveCheckPoint()
	if err != nil {
		log.Printf("failed to save ticket check point %v", err)
		return "", err
	}
	return "关闭成功", nil
}

func (t *MaimaiTicketMaster) NextRank(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if index >= int64(len(t.tickets)) {
		return "", fmt.Errorf("编号错误")
	}
	if index == -1 {
		for i, ticket := range t.tickets {
			if ticket.Creator == operator {
				index = int64(i)
				break
			}
		}
	}

	if !t.checkPermission(operator, index) {
		return "", fmt.Errorf("只能操作自己点的歌曲")
	}

	record, err := t.SmartPick(t.tickets[index].Keyword, t.tickets[index].Rank+1)
	if err != nil {
		return "", err
	}
	newTicket := &MaimaiTicket{
		Keyword: t.tickets[index].Keyword,
		Creator: t.tickets[index].Creator,
		Record:  record,
		Rank:    t.tickets[index].Rank + 1,
	}
	t.tickets[index] = newTicket
	err = t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "切换成功", nil
}

func (t *MaimaiTicketMaster) NextLevel(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if index >= int64(len(t.tickets)) {
		return "", fmt.Errorf("编号错误")
	}
	if index == -1 {
		for i, ticket := range t.tickets {
			if ticket.Creator == operator {
				index = int64(i)
				break
			}
		}
	}
	if !t.checkPermission(operator, index) {
		return "", fmt.Errorf("只能操作自己点的歌曲")
	}
	t.tickets[index].RotateLevel()
	err := t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "切换成功", nil
}

func NewMaimaiTicketMaster(songDatabasePath string, checkPointPath string,
	maxTicketSize int, maxAllowedTickets int) *MaimaiTicketMaster {
	t := &MaimaiTicketMaster{
		lock:                           sync.RWMutex{},
		maxTicketSize:                  maxTicketSize,
		maxAllowedTicketsForOneCreator: maxAllowedTickets,
		checkPointPath:                 checkPointPath,
		storage:                        NewMaimaiStorage(songDatabasePath),
	}

	if ok := t.loadCheckPoint(); ok != nil {
		t.tickets = make([]*MaimaiTicket, 0, maxTicketSize)
		err := t.saveCheckPoint()
		if err != nil {
			panic(err)
		}
	}

	return t
}

func (t *MaimaiTicketMaster) loadCheckPoint() error {
	if t.checkPointPath == "" {
		return nil
	}

	file, err := os.ReadFile(t.checkPointPath)
	if err != nil {
		return err
	}
	var tickets []*MaimaiTicket
	err = json.Unmarshal(file, &tickets)
	if err != nil {
		return err
	}
	t.tickets = tickets
	return nil
}

func (t *MaimaiTicketMaster) saveCheckPoint() error {
	if t.checkPointPath == "" {
		return nil
	}

	result, err := json.Marshal(t.tickets)
	if err != nil {
		return err
	}
	err = os.WriteFile(t.checkPointPath, result, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (t *MaimaiTicketMaster) AddTicket(creator string, keyword string) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(t.tickets) >= t.maxTicketSize {
		return "", errors.New("歌单已满~")
	}
	count := 0
	t.ForEachTicket(func(ticket model.ITicket) {
		if ticket.GetCreator() == creator {
			count++
		}
	})
	if count >= t.maxAllowedTicketsForOneCreator {
		return "", fmt.Errorf("已经点了%d首歌了，待会儿再点吧", t.maxAllowedTicketsForOneCreator)
	}

	record, err := t.SmartPick(keyword, 0)
	if err != nil {
		return "", err
	}

	t.tickets = append(t.tickets, &MaimaiTicket{
		Keyword: keyword,
		Creator: creator,
		Record:  record,
		Rank:    0,
	})
	err = t.saveCheckPoint()
	if err != nil {
		log.Printf("failed to save ticket check point %v", err)
		return "", err
	}
	return "成功！", nil
}

func (t *MaimaiTicketMaster) SmartPick(input string, rank int) (*MaimaiRecord, error) {
	if input == "" {
		return nil, errors.New("关键字不能为空")
	}
	specifiedTrackType := func(keyword string) (string, string) {
		trackTypeMapping := map[string]string{
			"标准":   "std",
			"标准谱":  "std",
			"STD":  "std",
			"std":  "std",
			"DX":   "dx",
			"dx":   "dx",
			"STD谱": "std",
			"std谱": "std",
			"DX谱":  "dx",
			"dx谱":  "dx",
		}
		for k, v := range trackTypeMapping {
			if strings.HasPrefix(keyword, k) || strings.HasSuffix(keyword, k) {
				return v, strings.Trim(keyword, k)
			}
		}

		return "", keyword
	}
	specifiedTrackLevel := func(keyword string) (int, string) {
		levelKey := []string{"白", "紫", "红", "绿", "黄"}
		for i, lk := range levelKey {
			if strings.HasPrefix(keyword, lk) {
				keyword = strings.TrimPrefix(keyword, lk)
				return i, keyword
			}
			if strings.HasSuffix(keyword, lk) {
				keyword = strings.TrimSuffix(keyword, lk)
				return i, keyword
			}
		}
		return -1, keyword
	}
	var trackType = ""
	var trackLevel int
	var keyword string

	trackType, keyword = specifiedTrackType(input)
	var record MaimaiRecord
	if t.storage.RankRecord(keyword)[0].score >= 100 {
		record = *t.storage.PickOneWithTrackType(keyword, rank, trackType)
	} else {
		trackLevel, keyword = specifiedTrackLevel(keyword)
		if trackType == "" {
			trackType, keyword = specifiedTrackType(keyword)
		}

		record = *t.storage.PickOneWithTrackType(keyword, 0, trackType)
		if trackLevel != -1 {
			record.CurrentLevel = len(record.Levels) - 5 + trackLevel
		}
	}

	return &record, nil
}

func (t *MaimaiTicketMaster) ForEachTicket(fn func(ticket model.ITicket)) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, ticket := range t.tickets {
		fn(ticket)
	}
	for i := 0; i < t.maxTicketSize-len(t.tickets); i++ {
		fn(&MaimaiTicket{
			Keyword: "",
			Creator: "-",
			Record: &MaimaiRecord{
				Rank:      0,
				Title:     "使用 点歌 <歌名>来自动匹配封面",
				Alias:     nil,
				ImagePath: "27254170d8811952baa1626557c101607b61bf526dcaf06491b71b0c416d315d.jpg",
				Levels: []MaimaiLevel{
					{
						Type:       "std",
						Difficulty: "",
						Level:      "bas",
					},
				},
				Category:     "等待选择",
				CurrentLevel: 0,
			},
			Rank: 0,
		})
	}
}
