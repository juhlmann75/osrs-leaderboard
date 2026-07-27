package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	defaultPostUsername = "Karwambwadan"
	defaultPostTime     = "09:00"
)

var (
	Token         string
	PostChannelID string
	PostUsername  string
	PostTime      string
)

func init() {
	Token = os.Getenv("BOT_TOKEN")
	PostChannelID = os.Getenv("POST_CHANNEL_ID")
	PostUsername = os.Getenv("POST_USERNAME")
	if PostUsername == "" {
		PostUsername = defaultPostUsername
	}
	PostTime = os.Getenv("POST_TIME")
	if PostTime == "" {
		PostTime = defaultPostTime
	}
}

type hiscoreResponse struct {
	Name   string         `json:"name"`
	Skills []hiscoreSkill `json:"skills"`
}

type hiscoreSkill struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Rank  int    `json:"rank"`
	Level int    `json:"level"`
	XP    int    `json:"xp"`
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

	// Start the daily agility post scheduler if a channel was configured.
	if PostChannelID != "" {
		go scheduleDailyPost(dg, PostChannelID, PostUsername, PostTime)
	} else {
		fmt.Println("POST_CHANNEL_ID not set; daily agility post disabled")
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
		_, after, found := strings.Cut(m.Content, " ")
		username := PostUsername
		if found {
			username = after
		}

		skill, err := getAgilityInfo(username)
		if err != nil {
			msg := "Error fetching OSRS hiscores: " + err.Error()
			if errors.Is(err, errInvalidUsername) {
				msg = "Invalid Username"
			}
			_, err = s.ChannelMessageSend(m.ChannelID, msg)
			if err != nil {
				fmt.Println(err)
			}
			return
		}

		_, err = s.ChannelMessageSend(m.ChannelID, formatAgility(skill, username))
		if err != nil {
			fmt.Println(err)
		}
	}
}

var errInvalidUsername = errors.New("invalid username")

// getAgilityInfo fetches the Agility skill entry for the given player from the
// OSRS hiscores JSON API.
func getAgilityInfo(username string) (*hiscoreSkill, error) {
	requestURL := "https://secure.runescape.com/m=hiscore_oldschool/index_lite.json?player=" + username
	response, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, errInvalidUsername
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from hiscores", response.StatusCode)
	}

	var hr hiscoreResponse
	if err := json.NewDecoder(response.Body).Decode(&hr); err != nil {
		return nil, err
	}

	for i := range hr.Skills {
		if hr.Skills[i].Name == "Agility" {
			return &hr.Skills[i], nil
		}
	}
	return nil, errors.New("agility skill not found")
}

// formatAgility renders the daily/message agility line.
func formatAgility(skill *hiscoreSkill, username string) string {
	return fmt.Sprintf("%s Agility — Rank: %s, **Level: %s**, XP: %s",
		username,
		strconv.Itoa(skill.Rank),
		strconv.Itoa(skill.Level),
		strconv.Itoa(skill.XP),
	)
}

// postAgility fetches the agility info for username and posts it to channelID.
// Errors are logged and swallowed so the bot stays alive.
func postAgility(s *discordgo.Session, channelID, username string) {
	skill, err := getAgilityInfo(username)
	if err != nil {
		fmt.Println("daily agility post failed:", err)
		return
	}
	_, err = s.ChannelMessageSend(channelID, formatAgility(skill, username))
	if err != nil {
		fmt.Println(err)
	}
}

// scheduleDailyPost posts the agility score once a day at postTime (HH:MM, local
// time). It does not fire on startup; if startup is after today's scheduled
// time, the next fire is tomorrow. Missed fires are skipped until the next day.
func scheduleDailyPost(s *discordgo.Session, channelID, username, postTime string) {
	hh, mm, ok := parseTime(postTime)
	if !ok {
		fmt.Printf("invalid POST_TIME %q; falling back to %s\n", postTime, defaultPostTime)
		hh, mm = 9, 0
	}

	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, time.Local)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}

	time.Sleep(time.Until(next))
	postAgility(s, channelID, username)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		postAgility(s, channelID, username)
	}
}

// parseTime parses an "HH:MM" string into hour and minute ints.
func parseTime(s string) (int, int, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}
