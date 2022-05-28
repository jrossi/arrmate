package server

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
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

func NewServer(ac *ArrConfig) (*ArrServer, error) {
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

	return as, nil
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
