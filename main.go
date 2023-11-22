package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	Token string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!leaderboard") {
		command := strings.Split(m.Content, " ")
		username := "D4N_K"
		if len(command) > 1 {
			username = command[1]
		}
		requestUrl := "https://secure.runescape.com/m=hiscore_oldschool_seasonal/index_lite.ws?player=" + username
		response, err := http.Get(requestUrl)
		if err != nil {
			fmt.Println(err)
		}
		defer response.Body.Close()

		if response.StatusCode == 200 {
			// Transform our response to a []byte
			body, err := io.ReadAll(response.Body)
			if err != nil {
				fmt.Println(err)
			}

			message := getMessage(body, username)

			_, err = s.ChannelMessageSend(m.ChannelID, message)
			if err != nil {
				fmt.Println(err)
			}
		} else if response.StatusCode == 404 {
			_, err = s.ChannelMessageSend(m.ChannelID, "Invalid Username")
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Error: Can't get OSRS info! :-(")
		}
	}
}

func getMessage(body []byte, username string) string {
	leaderboardContent := string(body)
	leaderboardContentSplit := strings.Split(leaderboardContent, "\n")
	leagueLeaderboard := leaderboardContentSplit[24]
	leagueInfo := strings.Split(leagueLeaderboard, ",")
	leagueRank := leagueInfo[0]
	leaguePoints := leagueInfo[1]
	message := username + " League Rank: " + leagueRank + ", Points: " + leaguePoints
	return message
}
