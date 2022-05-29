package main

import (
	"context"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/jrudio/go-plex-client"
	"github.com/rs/zerolog"
	"jeremyrossi.com/go/arrmate/server"
	"os"
	"zombiezen.com/go/sqlite/shell"
)

type grammer struct {
	/*
		Server struct {
		} `cmd:"server"  help:"server runs bot"`
		Config struct {
			ShellConfig struct {
			} `cmd:""  help:"config interacts with sql config store"`
			GetConfig struct {
			} `cmd:""  help:"config interaciiukkts with sql config store and returns a value for key"`
			SetConfig struct {
				Config map[string]float64 `arg:"" type:"file:"`
			} `cmd:""  help:"config interacts with sql config store and sets key=value"`
			ListConfig struct {
			} `cmd:"" help:"config interacts with sql config store and gets all keys"`
		}
	*/
	CS       string `name:"connectstring" default:"./arrmate.sqlite"`
	LogLevel string `name:"logging.level" default:"warn"`
	Server   struct {
	} `cmd`
	Config struct {
		Get struct {
			Values []string `arg:""`
		} `cmd`
		Set struct {
			Values map[string]string `arg:""`
		} `cmd`
		List struct {
		} `cmd`
		Shell struct {
		} `cmd`
	} `cmd`
	Plex struct {
		Test struct {
		} `cmd`
		Search struct {
			Value string `arg:""`
		} `cmd`
	} `cmd`
	Sonarr struct {
		Search struct {
			Value string `arg:""`
		} `cmd`
	} `cmd`
}

func (c *grammer) ConnectString() string {
	return c.CS
}

func (c *grammer) LoggingLevel() string {
	return c.LogLevel
}

func (g *grammer) SetupClient() (*server.ArrServer, error) {
	l, err := zerolog.ParseLevel(g.LoggingLevel())
	if err != nil {
		return nil, err
	}
	zerolog.SetGlobalLevel(l)
	return server.NewClient(g)

}

func HandlePlexSearch(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}
	//defer ac.Close()

	found, plexServer, err := ac.DB.ConfigGet("plex.url")
	if !found {
		return fmt.Errorf("No config for plex.url")
	} else if err != nil {
		return err
	}
	found, plexToken, err := ac.DB.ConfigGet("plex.token")
	if !found {
		return fmt.Errorf("No config for plex.token")
	} else if err != nil {
		return err
	}

	plexConn, err := plex.New(plexServer, plexToken)
	if err != nil {
		return err
	}

	fmt.Println("Searching for " + g.Plex.Search.Value)
	results, err := plexConn.Search(g.Plex.Search.Value)
	if err != nil {
		fmt.Println("plexConn test errored: ")
		return err
	}

	if len(results.MediaContainer.Metadata) == 0 {
		fmt.Println("could not find '" + g.Plex.Search.Value + "'")

		return nil
	}

	for _, searchResult := range results.MediaContainer.Metadata {
		fmt.Println(searchResult.Title)
	}
	return nil

}

func HandlePlexTest(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}
	result, err := ac.PlexConn.Test()
	if err != nil {
		return err
	}

	if !result {
		fmt.Println("failed to connect to plex")
		return nil
	}

	fmt.Println("successfully connected to plex")
	return nil

}

func HandleConfigSet(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}

	for k, v := range g.Config.Set.Values {
		err := ac.DB.ConfigSet(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func HandleConfigGet(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}
	//defer db.Close()

	for _, k := range g.Config.Get.Values {
		found, v, _ := ac.DB.ConfigGet(k)
		if err != nil {
			return err
		}
		if found {
			fmt.Printf("%s=%s\n", k, v)
		} else {
			fmt.Printf("%s=\n", k)
		}
	}
	return nil
}

func HandleConfigList(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}
	//defer db.Close()

	keys, _ := ac.DB.ConfigList()
	for _, k := range keys {
		found, v, _ := ac.DB.ConfigGet(k)
		if err != nil {
			return err
		}
		if found {
			fmt.Printf("%s=%s\n", k, v)
		} else {
			fmt.Printf("%s=\n", k)
		}
	}
	return nil
}
func HandleConfigShell(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}
	//defer db.Close()

	conn, err := ac.DB.Pool.Get(context.TODO())
	if err != nil {
		return err
	}
	defer conn.Close()
	shell.Run(conn)
	return nil
}

func HandleStarrSonarrSearch(g *grammer) error {
	ac, err := g.SetupClient()
	if err != nil {
		return err
	}
	//defer db.Close()
	ac.SearchSonarr(g.Sonarr.Search.Value)
	return nil

}

func StartServer(g *grammer) error {
	srv, err := server.NewServer(g)
	if err != nil {
		return err
	}
	return srv.Run()
}

func main() {
	g := &grammer{}
	ctx := kong.Parse(g,
		kong.Name("arrmate"),
		kong.Description("A discord bot for talking to other things"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))
	//fmt.Print(ctx.Command())
	var err error
	switch ctx.Command() {
	case "config set <values>":
		err = HandleConfigSet(g)
	case "config get <values>":
		err = HandleConfigGet(g)
	case "config list":
		err = HandleConfigList(g)
	case "config shell":
		err = HandleConfigShell(g)
	case "plex test":
		err = HandlePlexTest(g)
	case "plex search <value>":
		err = HandlePlexSearch(g)
	case "sonarr search <value>":
		err = HandleStarrSonarrSearch(g)
	case "server":
		err = StartServer(g)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(100)
	}
	/*
		switch ctx.Command() {
		case "rm <path>":
			fmt.Println(cli.Rm.Paths, cli.Rm.Force, cli.Rm.Recursive)

		case "ls":
		}
	*/

}
