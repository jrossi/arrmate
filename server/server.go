package server

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/go-co-op/gocron"
	"github.com/jrudio/go-plex-client"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type ArrServer struct {
	Session  *discordgo.Session
	PlexConn *plex.Plex
	DB       *DB
	Cron     *gocron.Scheduler
}

type ArrConfig struct {
	CS string `long:"connect" default:"" description:"SQLlite connect String"`
}

func (ac *ArrConfig) ConnectString() string {
	return ac.CS
}

func NewServer(ac DBConfig) (*ArrServer, error) {
	var err error

	db, err := NewDB(ac)
	if err != nil {
		return nil, err
	}

	as := &ArrServer{
		DB:   db,
		Cron: gocron.NewScheduler(time.UTC),
	}
	as.Cron.StartAsync()

	err = as.SetupDiscord()
	if err != nil {
		return nil, err
	}
	err = as.SetupPlex()
	if err != nil {
		return nil, err
	}
	err = as.SetupStarr()
	if err != nil {
		return nil, err
	}

	return as, nil
}

func NewClient(ac DBConfig) (*ArrServer, error) {
	var err error
	db, err := NewDB(ac)
	if err != nil {
		return nil, err
	}

	as := &ArrServer{
		DB: db,
	}
	err = as.SetupPlex()
	if err != nil {
		return nil, err
	}

	return as, nil
}

func (srv *ArrServer) SetupPlex() error {

	found, plexServer, err := srv.DB.ConfigGet("plex.url")
	if !found {
		return fmt.Errorf("No config for plex.url")
	} else if err != nil {
		return err
	}
	found, plexToken, err := srv.DB.ConfigGet("plex.token")
	if !found {
		return fmt.Errorf("No config for plex.token")
	} else if err != nil {
		return err
	}

	plexConn, err := plex.New(plexServer, plexToken)
	if err != nil {
		return err
	}
	srv.PlexConn = plexConn

	result, err := srv.PlexConn.Test()
	if err != nil {
		return err
	}

	if !result {
		log.Warn().Str("src", "server.plex").Msg("plexConn.Test() did not return results")
		return nil
	}

	log.Debug().Str("src", "server.plex").Msg("SetupPlex completed")
	return nil
}

func (srv *ArrServer) SetupDiscord() error {
	found, token, err := srv.DB.ConfigGet("discord.token")
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("discord.token must be set before starting the arrmate server")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}
	srv.Session = s
	srv.Session.Identify.Intents = discordgo.IntentsGuildMessages
	srv.Session.AddHandler(srv.OnReady)
	srv.Session.AddHandler(srv.DiscordMessageHandler)

	return nil

}

func (srv *ArrServer) Run() error {
	err := srv.Session.Open()
	if err != nil {
		//return fmt.Println("Error opening Discord session: ", err)
		return err
	}

	guilds, err := srv.Session.UserGuilds(100, "", "")
	if len(guilds) == 0 {
		fmt.Print("\t(none)")
	}
	for index := range guilds {
		guild := guilds[index]
		fmt.Print("\t", guild.Name, " (", guild.ID, ")")
	}
	//fmt.Print("channel name: ", activeChannel)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("arrmate is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	return srv.Session.Close()
}

// OnReady handles the "ready" event from Discord
func (srv *ArrServer) OnReady(s *discordgo.Session, e *discordgo.Ready) {
	fmt.Println("Session ready")
}

func (srv *ArrServer) DiscordMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	//fmt.Println(m.Content)

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		srv.HandlePing(s, m)
	}
	if strings.HasPrefix(m.Content, "!plex search ") {
		srv.HandlePlexSearch(s, m)
	}
	if strings.HasPrefix(m.Content, "!sonarr search ") {
		srv.HandleSonarrSearch(s, m)
	}

	/*
			if strings.HasPrefix(m.Content, "!sql ") {
				srv.HandleSQL(s, m)
			}
			if strings.HasPrefix(m.Content, "!config ") {
				srv.HandleSQL(s, m)
			}

		if strings.HasPrefix(m.Content, "!plex ") {
			srv.HandlePlex(s, m)
		}
	*/

}

func (srv *ArrServer) HandlePing(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Pong!")
}
func (srv *ArrServer) HandlePlexSearch(s *discordgo.Session, m *discordgo.MessageCreate) {
	ss := strings.TrimPrefix(m.Content, "!plex search ")
	//fmt.Println("--->" + ss + "<---")
	//fmt.Println("--->" + srv.PlexConn.URL + "<---")
	results, err := srv.PlexConn.Search(ss)
	if err != nil {
		log.Warn().Err(err).Str("search", ss).Err(err).Msg("Problem with user search")
		return
	}
	if len(results.MediaContainer.Metadata) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Could not find results with Search: "+ss)
		return
	}

	var b bytes.Buffer

	for _, searchResult := range results.MediaContainer.Metadata {
		b.WriteString(searchResult.Title)
		b.WriteString("\n")
	}
	s.ChannelMessageSend(m.ChannelID, b.String())

}
