//go:build sqlite_vtable || vtable
// +build sqlite_vtable vtable

package vtables

import (
	"fmt"
	"github.com/mattn/go-sqlite3"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

type SonarrModele struct{}

type SonarrTableSeries struct {
	sonarr.Sonarr
}

func (sm *SonarrModele) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {

	sonarr.Series{}
	scfg := starr.New()
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
			id INT,
			title TEXT,
			status TEXT,
			Overview TEXT,
			PreviousAiring    
			Network           string  
	Year              int              
	Path              string          
	QualityProfileID  int64          
	LanguageProfileID int64         
	Runtime           int          
	TvdbID            int64       
	TvRageID          int64      
	TvMazeID          int64     
	FirstAired        time.Time 
	SeriesType        string   
	CleanTitle        string  
	ImdbID            string  
	TitleSlug         string  
	RootFolderPath    string  
	Certification     string  
	Genres            []string  
	Tags              []int    
	Added             time.Time 
	NextAiring        time.Time 
	AirTime           string   
	Ended             bool    
	SeasonFolder      bool   
	Monitored         bool              `json:"monitored"`
	UseSceneNumbering bool              `json:"useSceneNumbering,omitempty"`
	Images            []*starr.Image    `json:"images,omitempty"`
	Seasons           []*Season         `json:"seasons,omitempty"`
	Statistics        *Statistics       `json:"statistics,omitempty"`
	Ratings           *starr.Ratings    `json:"ratings,omitempty"`

			

			Monitored INT
		)`, args[0]))
	if err != nil {
		return nil, err
	}
	return &ghRepoTable{}, nil
}

func (m *SonarrModele) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}
