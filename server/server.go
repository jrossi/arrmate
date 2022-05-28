package server

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"syscall"
)

type ArrServer struct {
	Session *discordgo.Session
	DB      *DB
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
		DB: db,
	}

	found, token, err := db.ConfigGet("discord.token")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("discord.token must be set before starting the arrmate server")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	as.Session = s
	as.Session.Identify.Intents = discordgo.IntentsGuildMessages

	return as, nil
}

func (srv *ArrServer) Run() error {
	err := srv.Session.Open()
	if err != nil {
		//return fmt.Println("Error opening Discord session: ", err)
		return err
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("arrmate is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	return srv.Session.Close()
}

func (srv *ArrServer) DiscordMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		srv.HandlePing(s, m)
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
