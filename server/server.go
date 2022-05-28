package server

import "github.com/bwmarrin/discordgo"

type ArrServer struct {
	Session *discordgo.Session
}

type ArrConfig struct {
	Token string
}

func New(ac *ArrConfig) (*ArrServer, error) {
	var err error
	s, err := discordgo.New("Bot " + ac.Token)
	if err != nil {
		return nil, err
	}

	as := &ArrServer{
		Session: s,
	}
	return as, nil
}
