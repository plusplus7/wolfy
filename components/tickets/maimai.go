package tickets

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"wolfy/components/songs"
	"wolfy/internal/fileutil"
	"wolfy/model"
)

type MaimaiTicket struct {
	Keyword string              `json:"keyword"`
	Creator string              `json:"creator"`
	Record  *songs.MaimaiRecord `json:"record"`
	Rank    int                 `json:"rank"`
	Level   int                 `json:"level"`
}

func (m *MaimaiTicket) RotateLevel() {
	m.Level++
}

func (m *MaimaiTicket) GetKeyword() string {
	return m.Keyword
}

func (m *MaimaiTicket) GetCoverPath() string {
	if m == nil || m.Record == nil {
		return ""
	}
	return m.Record.ImagePath
}

func (m *MaimaiTicket) GetCoverInfo() string {
	if m == nil || m.Record == nil {
		return "-"
	}
	return m.Record.GetTrackType(m.Level)
}

func (m *MaimaiTicket) GetGenreInfo() string {
	if m == nil || m.Record == nil {
		return "-"
	}
	return m.Record.Category
}

func (m *MaimaiTicket) GetSongInfo() string {
	if m == nil || m.Record == nil {
		return "-_-"
	}
	return m.Record.GetTrackLevel(m.Level) + "_" + m.Record.GetTrackDifficulty(m.Level)
}

func (m *MaimaiTicket) GetTitle() string {
	if m == nil || m.Record == nil {
		return "-"
	}
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
	storage        *songs.MaimaiStorage
}

const (
	superAdmin           = "主播"
	maxTicketsPerCreator = 3
)

func (t *MaimaiTicketMaster) checkPermission(creator string, index int64) bool {
	if !t.validIndex(index) {
		return false
	}
	return t.tickets[index].Creator == creator || creator == superAdmin
}

func (t *MaimaiTicketMaster) validIndex(index int64) bool {
	return index >= 0 && index < int64(len(t.tickets))
}

func (t *MaimaiTicketMaster) countTicketsByCreator(creator string) int {
	count := 0
	for _, ticket := range t.tickets {
		if ticket != nil && ticket.Creator == creator {
			count++
		}
	}
	return count
}

func (t *MaimaiTicketMaster) resolveIndex(operator string, index int64) (int64, error) {
	if index == -1 {
		for i, ticket := range t.tickets {
			if ticket != nil && ticket.Creator == operator {
				return int64(i), nil
			}
		}
		return -1, fmt.Errorf("%s 找不到错误", operator)
	}
	if !t.validIndex(index) {
		return -1, fmt.Errorf("%s 下标错误", operator)
	}
	return index, nil
}

func (t *MaimaiTicketMaster) FinishTicket(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	index, err := t.resolveIndex(operator, index)
	if err != nil {
		return "", err
	}

	if !t.checkPermission(operator, index) {
		return "", fmt.Errorf("%s 只能操作自己点的歌曲", operator)
	}
	t.tickets = append(t.tickets[:index], t.tickets[index+1:]...)
	err = t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "关闭成功", nil
}

func (t *MaimaiTicketMaster) NextRank(operator string, index int64) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	index, err := t.resolveIndex(operator, index)
	if err != nil {
		return "", err
	}

	if !t.checkPermission(operator, index) {
		return "", fmt.Errorf("%s 只能操作自己点的歌曲", operator)
	}

	newTicket := &MaimaiTicket{
		Keyword: t.tickets[index].Keyword,
		Creator: t.tickets[index].Creator,
		Rank:    t.tickets[index].Rank + 1,
		Level:   t.tickets[index].Level,
	}
	newTicket.Record, err = t.storage.PickOne(t.tickets[index].Keyword, t.tickets[index].Rank+1)
	if err != nil {
		return "", err
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
	index, err := t.resolveIndex(operator, index)
	if err != nil {
		return "", err
	}
	if !t.checkPermission(operator, index) {
		return "", fmt.Errorf("%s 只能操作自己点的歌曲", operator)
	}
	t.tickets[index].RotateLevel()
	err = t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "切换成功", nil
}

func NewMaimaiTicketMaster(songDatabasePath string, aliasFilePath string, checkPointPath string,
	maxTicketSize int) *MaimaiTicketMaster {
	return NewMaimaiTicketMasterWithStorage(songs.NewMaimaiStorage(songDatabasePath, aliasFilePath), checkPointPath, maxTicketSize)
}

func NewMaimaiTicketMasterWithStorage(storage *songs.MaimaiStorage, checkPointPath string, maxTicketSize int) *MaimaiTicketMaster {
	t := &MaimaiTicketMaster{
		lock:           sync.RWMutex{},
		maxTicketSize:  maxTicketSize,
		checkPointPath: checkPointPath,
		storage:        storage,
	}

	if err := t.loadCheckPoint(); err != nil {
		t.tickets = make([]*MaimaiTicket, 0, maxTicketSize)
		if errors.Is(err, os.ErrNotExist) {
			if err := t.saveCheckPoint(); err != nil {
				log.Printf("failed to initialize ticket checkpoint: %v", err)
			}
		} else {
			log.Printf("failed to load ticket checkpoint: %v", err)
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
	return fileutil.AtomicWriteFile(t.checkPointPath, result, 0644)
}

func (t *MaimaiTicketMaster) AddTicket(creator string, keyword string) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(t.tickets) >= t.maxTicketSize {
		return "", errors.New("歌单已满~")
	}
	if t.countTicketsByCreator(creator) >= maxTicketsPerCreator {
		return "", fmt.Errorf("%s 每人限点%d首", creator, maxTicketsPerCreator)
	}
	targetLevel := 0
	if strings.HasSuffix(keyword, "紫") || strings.HasPrefix(keyword, "紫") {
		keyword = strings.Trim(keyword, "紫")
		targetLevel = -4
	} else if strings.HasSuffix(keyword, "红") || strings.HasPrefix(keyword, "红") {
		keyword = strings.Trim(keyword, "红")
		targetLevel = -3
	}
	record, err := t.storage.PickOne(keyword, 0)
	if err != nil {
		return "", err
	}
	t.tickets = append(t.tickets, &MaimaiTicket{
		Keyword: keyword,
		Creator: creator,
		Record:  record,
		Rank:    0,
		Level:   targetLevel,
	})
	err = t.saveCheckPoint()
	if err != nil {
		return "", err
	}
	return "成功！", nil
}

func (t *MaimaiTicketMaster) ForEachTicket(fn func(ticket model.ITicket)) {
	t.lock.RLock()
	tickets := make([]*MaimaiTicket, len(t.tickets))
	copy(tickets, t.tickets)
	maxTicketSize := t.maxTicketSize
	t.lock.RUnlock()

	for _, ticket := range tickets {
		fn(ticket)
	}
	for i := 0; i < maxTicketSize-len(tickets); i++ {
		fn(&MaimaiTicket{
			Keyword: "",
			Creator: "-",
			Record: &songs.MaimaiRecord{
				Title:     "使用 点歌 <歌名>来自动匹配封面",
				ImagePath: "https://assets2.lxns.net/maimai/jacket/1444.png",
				Levels: []songs.MaimaiLevel{
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
