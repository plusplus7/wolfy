package service

import "encoding/xml"

type MusicData struct {
	XMLName        xml.Name   `xml:"MusicData"`
	DataName       string     `xml:"dataName"`
	NetOpenName    IDString   `xml:"netOpenName"`
	ReleaseTagName IDString   `xml:"releaseTagName"`
	Disable        bool       `xml:"disable"`
	LongMusic      int        `xml:"longMusic"`
	Name           IDString   `xml:"name"`
	RightsInfoName IDString   `xml:"rightsInfoName"`
	SortName       string     `xml:"sortName"`
	ArtistName     IDString   `xml:"artistName"`
	GenreName      IDString   `xml:"genreName"`
	BPM            int        `xml:"bpm"`
	Version        int        `xml:"version"`
	AddVersion     IDString   `xml:"AddVersion"`
	MovieName      IDString   `xml:"movieName"`
	CueName        IDString   `xml:"cueName"`
	Dresscode      bool       `xml:"dresscode"`
	EventName      IDString   `xml:"eventName"`
	EventName2     IDString   `xml:"eventName2"`
	SubEventName   IDString   `xml:"subEventName"`
	LockType       int        `xml:"lockType"`
	SubLockType    int        `xml:"subLockType"`
	DotNetListView bool       `xml:"dotNetListView"`
	NotesData      NotesData  `xml:"notesData"`
	UtageKanjiName string     `xml:"utageKanjiName"`
	Comment        string     `xml:"comment"`
	UtagePlayStyle int        `xml:"utagePlayStyle"`
	FixedOptions   []FixedOpt `xml:"fixedOptions>FixedOption"`
	JacketFile     string     `xml:"jacketFile"`
	ThumbnailName  string     `xml:"thumbnailName"`
	RightFile      string     `xml:"rightFile"`
	Priority       int        `xml:"priority"`
}

type IDString struct {
	ID  int    `xml:"id"`
	Str string `xml:"str"`
}

type NotesData struct {
	Notes []Notes `xml:"Notes"`
}

type Notes struct {
	File struct {
		Path string `xml:"path"`
	} `xml:"file"`
	Level         int      `xml:"level"`
	LevelDecimal  int      `xml:"levelDecimal"`
	NotesDesigner IDString `xml:"notesDesigner"`
	NotesType     int      `xml:"notesType"`
	MusicLevelID  int      `xml:"musicLevelID"`
	MaxNotes      int      `xml:"maxNotes"`
	IsEnable      bool     `xml:"isEnable"`
}

type FixedOpt struct {
	FixedOptionName  string `xml:"_fixedOptionName"`
	FixedOptionValue string `xml:"_fixedOptionValue"`
}
