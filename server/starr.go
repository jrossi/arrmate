package server

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"golift.io/starr"
	"golift.io/starr/sonarr"
	"strconv"
	"strings"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (srv *ArrServer) SetupStarr() error {
	job, err := srv.Cron.Every("5m").StartImmediately().Do(func() {
		srv.BuildSonarr()
	})
	job.Tag("sonarr", "sonarr.starr")
	return err
}

func FormatBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (srv *ArrServer) BuildSonarr() error {
	found, url, err := srv.DB.ConfigGet("starr.sonarr.url")
	if !found {
		return fmt.Errorf("No config for plex.url")
	} else if err != nil {
		return err
	}
	found, token, err := srv.DB.ConfigGet("starr.sonarr.token")
	if !found {
		return fmt.Errorf("No config for plex.token")
	} else if err != nil {
		return err
	}
	scfg := starr.New(token, url, 1000000000)
	scfg.Debugf = log.Debug().Msgf

	s := sonarr.New(scfg)
	//return s.Lookup(ss)
	results, err := s.GetAllSeries()
	if err != nil {
		return err
	}

	conn, err := srv.DB.Pool.Get(context.TODO())
	if err != nil {
		return err
	}

	doUpdate := func() (err error) {
		defer sqlitex.Save(conn)(&err)

		err = sqlitex.Execute(conn, "DELETE from sonarr;", nil)
		if err != nil {
			return fmt.Errorf("database: %w", err)
		}
		q := `INSERT INTO sonarr (id, title, status, overview, previous_airing, network, added, genres, seasons, monitored) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
		for i, s := range results {
			log.Debug().Int("range item", i).Msg("populating database item")
			err = sqlitex.Execute(conn, q, &sqlitex.ExecOptions{
				Args: []interface{}{
					s.ID,
					s.Title,
					s.Status,
					s.Overview,
					s.PreviousAiring.Format("2006-01-02"),
					s.Network,
					s.Added.Format("2006-01-02"),
					strings.Join(s.Genres, ","),
					len(s.Seasons),
					FormatBool(s.Monitored),
				},
			})
			if err != nil {
				return err
			}
		}

		return nil
	}
	return doUpdate()
}

func (srv *ArrServer) HandleSonarrSearch(s *discordgo.Session, m *discordgo.MessageCreate) {
	ss := strings.TrimPrefix(m.Content, "!sonarr search ")
	if !strings.HasPrefix(ss, "%") {
		ss = fmt.Sprintf("%%%s", ss)
	}
	if !strings.HasSuffix(ss, "%") {
		ss = fmt.Sprintf("%s%%", ss)
	}
	log.Debug().Str("sonarr", "search").Str("query", ss).Msg("Sonarr Query log")

	conn, err := srv.DB.Pool.Get(context.TODO())
	if err != nil {
		log.Error().Err(err).Msg("Database could connected had a issue")
		return
	}

	var b bytes.Buffer
	err = sqlitex.Execute(conn, "SELECT id, title, status, previous_airing, added, seasons, monitored FROM sonarr WHERE title LIKE ?", &sqlitex.ExecOptions{
		Args: []interface{}{ss},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			log.Debug().Int64("id", stmt.ColumnInt64(0)).Msg("ResultsFunc logging - entry found")
			b.WriteString("id=")
			b.WriteString(strconv.FormatInt(stmt.ColumnInt64(0), 10))
			b.WriteString(" title=")
			b.WriteString(stmt.ColumnText(1))
			b.WriteString(" status=")
			b.WriteString(stmt.ColumnText(2))
			b.WriteString(" list=")
			b.WriteString(stmt.ColumnText(3))
			b.WriteString(" added=")
			b.WriteString(stmt.ColumnText(4))
			b.WriteString("\n")
			if b.Len() >= 1500 {
				_, err := s.ChannelMessageSend(m.ChannelID, b.String())
				if err != nil {
					log.Error().Err(err).Msg("Sending message failed")
				}
				b.Reset()
			}
			return nil

		},
	})
	_, err = s.ChannelMessageSend(m.ChannelID, b.String())
	if err != nil {
		log.Error().Err(err).Msg("Sending message failed")
	}

}

/*
func (srv *ArrServer) HandleSonarrSearch(s *discordgo.Session, m *discordgo.MessageCreate) {
	ss := strings.TrimPrefix(m.Content, "!sonarr search ")
	fmt.Print(ss)
	results, err := srv.SearchSonarr(ss)
	if err != nil {
		log.Warn().Err(err).Str("search", ss).Err(err).Msg("Problem with user search")
		return
	}
	if len(results) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Could not find results with Search: "+ss)
		return
	}

	var b bytes.Buffer
	for i, item := range results {
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". id=")
		b.WriteString(strconv.FormatInt(item.TvdbID, 10))
		b.WriteString(" title=")
		b.WriteString(item.Title)
		b.WriteString(" monitored=")
		b.WriteString(strconv.FormatBool(item.Monitored))
		b.WriteString(" status=")
		b.WriteString(item.Status)
		b.WriteString("\n")
		if i > 10 {
			break
		}
	}
	s.ChannelMessageSend(m.ChannelID, b.String())

}
*/
