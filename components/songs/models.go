package songs

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

type MaimaiStorageCache struct {
	Version int                   `json:"version"`
	Records map[int]*MaimaiRecord `json:"records"`
	Aliases map[int][]string      `json:"aliases"`
}

func (r *MaimaiRecord) GetTrackType(level int) string {
	track := r.trackAt(level)
	if track == nil {
		return "-"
	}
	return track.Type
}

func (r *MaimaiRecord) GetTrackLevel(level int) string {
	track := r.trackAt(level)
	if track == nil {
		return "-"
	}
	return track.Level
}

func (r *MaimaiRecord) GetTrackDifficulty(level int) string {
	track := r.trackAt(level)
	if track == nil {
		return "-"
	}
	return track.Difficulty
}

func (r *MaimaiRecord) trackAt(level int) *MaimaiLevel {
	if r == nil || len(r.Levels) == 0 {
		return nil
	}
	index := (2*len(r.Levels) - 1 - level) % len(r.Levels)
	if index < 0 {
		index += len(r.Levels)
	}
	return &r.Levels[index]
}
