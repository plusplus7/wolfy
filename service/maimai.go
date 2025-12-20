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

type Aliases struct {
	Alias []Alias `json:"aliases"`
}
type Alias struct {
	SongID  int      `json:"song_id"`
	Aliases []string `json:"aliases"`
}

type MaimaiLevel struct {
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
	Level      string `json:"level"`
}

type MaimaiRecord struct {
	ID        int           `json:"id"`
	Title     string        `json:"title"`
	ImagePath string        `json:"image"`
	Levels    []MaimaiLevel `json:"levels"`
	Category  string        `json:"category"`
}

func (r *MaimaiRecord) GetTrackType(level int) string {
	return r.Levels[(2*len(r.Levels)-1-level)%len(r.Levels)].Type
}

func (r *MaimaiRecord) GetTrackLevel(level int) string {
	return r.Levels[(2*len(r.Levels)-1-level)%len(r.Levels)].Level
}

func (r *MaimaiRecord) GetTrackDifficulty(level int) string {
	return r.Levels[(2*len(r.Levels)-1-level)%len(r.Levels)].Difficulty
}

type MaimaiTicket struct {
	Keyword string        `json:"keyword"`
	Creator string        `json:"creator"`
	Record  *MaimaiRecord `json:"record"`
	Rank    int           `json:"rank"`
	Level   int           `json:"level"`
}

func (m *MaimaiTicket) RotateLevel() {
	m.Level++
}

func (m *MaimaiTicket) GetKeyword() string {
	return m.Keyword
}

func (m *MaimaiTicket) GetCoverPath() string {
	return m.Record.ImagePath
}

func (m *MaimaiTicket) GetCoverInfo() string {
	return m.Record.GetTrackType(m.Level)
}

func (m *MaimaiTicket) GetGenreInfo() string {
	return m.Record.Category
}

func (m *MaimaiTicket) GetSongInfo() string {
	return m.Record.GetTrackLevel(m.Level) + "_" + m.Record.GetTrackDifficulty(m.Level)
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

	maxTicketSize  int
	checkPointPath string
	storage        *MaimaiStorage
}

const (
	superAdmin = "主播"
)

func (t *MaimaiTicketMaster) checkPermission(creator string, index int64) bool {
	if index >= int64(len(t.tickets)) {
		return false
	}
	return t.tickets[index].Creator == creator || creator == superAdmin
}

func (t *MaimaiTicketMaster) FinishTicket(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if index >= int64(len(t.tickets)) {
		return "", fmt.Errorf("%s 编号错误", operator)
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
		return "", fmt.Errorf("%s 只能操作自己点的歌曲", operator)
	}
	t.tickets = append(t.tickets[:index], t.tickets[index+1:]...)
	err := t.saveCheckPoint()
	if err != nil {
		log.Fatalf("failed to save ticket check point %v", err)
		return "", err
	}
	return "关闭成功", nil
}

func (t *MaimaiTicketMaster) NextRank(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if index >= int64(len(t.tickets)) {
		return "", fmt.Errorf("%s 编号错误", operator)
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
		return "", fmt.Errorf("%s 只能操作自己点的歌曲", operator)
	}

	newTicket := &MaimaiTicket{
		Keyword: t.tickets[index].Keyword,
		Creator: t.tickets[index].Creator,
		Record:  t.storage.PickOne(t.tickets[index].Keyword, t.tickets[index].Rank+1),
		Rank:    t.tickets[index].Rank + 1,
		Level:   t.tickets[index].Level,
	}
	t.tickets[index] = newTicket
	err := t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "切换成功", nil
}

func (t *MaimaiTicketMaster) NextLevel(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if index >= int64(len(t.tickets)) {
		return "", fmt.Errorf("%s 编号错误", operator)
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
		return "", fmt.Errorf("%s 只能操作自己点的歌曲", operator)
	}
	fmt.Println(t.tickets[index])
	t.tickets[index].RotateLevel()
	fmt.Println(t.tickets[index])
	err := t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "切换成功", nil
}

func NewMaimaiTicketMaster(songDatabasePath string, aliasFilePath string, checkPointPath string,
	maxTicketSize int) *MaimaiTicketMaster {
	t := &MaimaiTicketMaster{
		lock:           sync.RWMutex{},
		maxTicketSize:  maxTicketSize,
		checkPointPath: checkPointPath,
		storage:        NewMaimaiStorage(songDatabasePath, aliasFilePath),
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
	targetLevel := 0
	if strings.HasSuffix(keyword, "紫") || strings.HasPrefix(keyword, "紫") {
		keyword = strings.Trim(keyword, "紫")
		targetLevel = -4
	} else if strings.HasSuffix(keyword, "红") || strings.HasPrefix(keyword, "红") {
		keyword = strings.Trim(keyword, "红")
		targetLevel = -3
	}
	t.tickets = append(t.tickets, &MaimaiTicket{
		Keyword: keyword,
		Creator: creator,
		Record:  t.storage.PickOne(keyword, 0),
		Rank:    0,
		Level:   targetLevel,
	})
	err := t.saveCheckPoint()
	if err != nil {
		log.Fatalf("failed to save ticket check point %v", err)
		return "", err
	}
	return "成功！", nil
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
				Title:     "使用 点歌 <歌名>来自动匹配封面",
				ImagePath: "https://assets2.lxns.net/maimai/jacket/1444.png",
				Levels: []MaimaiLevel{
					{
						Type:       "std",
						Difficulty: "bas",
						Level:      "",
					},
				},
				Category: "等待选择",
			},
			Rank:  0,
			Level: 0,
		})
	}
}
