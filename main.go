package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"goDiscInfoBot/cwlog"
)

var (
	bootup       time.Time
	skipThrottle bool

	helpsFile string
)

func main() {
	bootup = time.Now()

	token := flag.String("token", "", "discord token")
	role := flag.String("staffid", "", "discord role ID for moderator/staff")
	guildid := flag.String("guildid", "", "discord guild id")
	testMode := flag.Bool("testmode", false, "skip throttle check")
	helpPath := flag.String("helpFilePath", "helps.hlp", "Specify path to helps file.")
	flag.Parse()

	discToken = *token
	staffRole = *role
	guildID = *guildid
	skipThrottle = *testMode
	helpsFile = *helpPath

	/* Start cw logs */
	cwlog.StartCWLog()
	cwlog.DoLog("Starting goDiscInfoBot.")

	readHelps()
	go CheckLife()
	go startbot()

	/* Wait here for process signals */
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	writeHelps()
}

func startbot() {

	/* Check if Discord token is set */
	if discToken == "" {
		cwlog.DoLog("Discord token not set, not starting.")
		return
	}

	/* Attempt to start bot */
	cwlog.DoLog("Starting Discord bot...")
	bot, erra := discordgo.New("Bot " + discToken)

	/*
	 * If we fail, keep attempting with increasing delay and maximum tries
	 * We do this, in case there is a failure.
	 * Discord will invalidate the token if there are too many connection attempts.
	 */
	if erra != nil {
		cwlog.DoLog(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %v", erra))
		time.Sleep(time.Duration(discordConnectAttempts*5) * time.Second)
		discordConnectAttempts++

		if discordConnectAttempts < maxAttempts {
			startbot()
		}
		return
	}

	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged)

	/* This is called when the connection is verified */
	bot.AddHandler(BotReady)
	errb := bot.Open()

	/* This handles error after the inital connection */
	if errb != nil {
		cwlog.DoLog(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %v", errb))
		time.Sleep(time.Duration(discordConnectAttempts*5) * time.Second)
		discordConnectAttempts++

		if discordConnectAttempts < maxAttempts {
			startbot()
		}
		return
	}

	/* This drastically reduces log spam */
	bot.LogLevel = discordgo.LogWarning
}

func BotReady(s *discordgo.Session, r *discordgo.Ready) {

	/* Set the bot's Discord status message */
	botstatus := "m45sci.xyz"
	errc := s.UpdateGameStatus(0, botstatus)
	if errc != nil {
		cwlog.DoLog(errc.Error())
	}

	/* Message and command hooks */
	s.AddHandler(MessageCreate)

	if s != nil {
		/* Save Discord descriptor, we need it */
		ds = s
	}

	cwlog.DoLog("Discord bot ready.")

	//Reset attempt count, we are fully connected.
	discordConnectAttempts = 0
}

func CheckLife() {
	for {
		time.Sleep(time.Hour)
		if time.Since(bootup) > rebootTime {
			os.Exit(0)
		}
	}
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	/* Ignore messages from self */
	if m.Author.ID == s.State.User.ID {
		return
	}

	/* Throw away messages from bots */
	if m.Author.Bot {
		return
	}

	if m.GuildID != guildID {
		fmt.Println("Incorrect guild: " + m.Member.GuildID)
		return
	}

	filterMessages(s, m)
}

func filterMessages(s *discordgo.Session, m *discordgo.MessageCreate) {
	resposeCount := 0
	respondedTo := map[int]bool{}
	staffMode := false

	//Switch lists if user is staff
	searchList := HelpsListData{}

	if staffRole != "" {
		for _, role := range m.Member.Roles {
			if role == staffRole {
				staffMode = true
				break
			}
		}
	}

	for _, help := range helpsList {
		if !staffMode && strings.EqualFold(help.Name, "main") {
			searchList = help
			break
		} else if staffMode && strings.EqualFold(help.Name, "staff") {
			searchList = help
			break
		}
	}

	if len(searchList.Data) == 0 {
		cwlog.DoLog("No helps data found.")
		return
	}

	//Condense the string for wildcard matches
	msgLower := strings.ToLower(m.Content)
	msgWild := strings.ReplaceAll(msgLower, " the ", " ")
	msgWild = strings.ReplaceAll(msgWild, " ", "")
	msgWild = strings.ReplaceAll(msgWild, "-", "")

	outLines := []string{}

	caser := cases.Title(language.AmericanEnglish)

	for h, help := range searchList.Data {
		if respondedTo[h] {
			continue
		}
		if resposeCount >= maxCombinedResponses {
			break
		}
		for _, searchWild := range help.Wildcards {
			if respondedTo[h] {
				continue
			}
			if resposeCount >= maxCombinedResponses {
				break
			}
			if strings.Contains(msgWild, searchWild) {
				doExclude := false
				for _, exclude := range help.Exclude {
					if strings.Contains(msgWild, exclude) {
						doExclude = true
						break
					}
				}
				if doExclude {
					doExclude = false
				} else {
					if respondedTo[h] {
						continue
					}
					if len(outLines) != 0 {
						outLines = append(outLines, "")
					}
					outLines = append(outLines, caser.String(searchWild+": "))
					outLines = append(outLines, help.ReplyLines...)
					resposeCount++
					respondedTo[h] = true
				}
			}
		}

		msgWords := strings.Split(msgLower, " ")
		for _, msgWord := range msgWords {
			if respondedTo[h] {
				continue
			}
			if resposeCount >= maxCombinedResponses {
				break
			}
			for _, helpWord := range help.Words {
				if strings.Contains(msgWord, helpWord) {
					doExclude := false
					for _, exclude := range help.Exclude {
						if strings.Contains(msgWild, exclude) {
							doExclude = true
							break
						}
					}
					if doExclude {
						doExclude = false
					} else {
						if len(outLines) != 0 {
							outLines = append(outLines, "")
						}
						outLines = append(outLines, caser.String(helpWord+": "))
						outLines = append(outLines, help.ReplyLines...)
						resposeCount++
						respondedTo[h] = true
					}
				}
			}
		}
	}

	if len(outLines) > 0 {
		if checkThrottle(s, m) {
			cwlog.DoLog(fmt.Sprintf("TRIGGERED:\n%v: %v: %v\nReply: %v", m.ChannelID, m.Author.Username, m.Content, strings.Join(outLines, "\n")))
			SmartWriteDiscord(m.ChannelID, strings.Join(outLines, "\n"))
		}
	}
}

/*Send normal message to a channel*/
func SmartWriteDiscord(ch string, text string) {

	if ch == "" || text == "" {
		return
	}

	if ds != nil {
		_, err := ds.ChannelMessageSend(ch, text)

		if err != nil {

			cwlog.DoLog(fmt.Sprintf("SmartWriteDiscord: ERROR: %v", err))
		}
	}
}

func checkThrottle(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if skipThrottle {
		return true
	}

	if totalMsgCount > maxGlobal {
		return false
	}

	if time.Since(lastReply) < throttleGlobal {
		cwlog.DoLog(fmt.Sprintf("global throttled: User: %v, Message: %v", m.Author.ID, m.Content))
		return false
	}
	if users[m.Author.ID] == nil {
		users[m.Author.ID] = &userData{id: m.Author.ID, lastSaw: time.Now()}
	} else {
		if users[m.Author.ID].total > maxPerUser {
			return false
		}
		if time.Since(users[m.Author.ID].lastSaw) < throttlePerUser {
			cwlog.DoLog(fmt.Sprintf("user throttled: User: %v, Message: %v", m.Author.ID, m.Content))
			return false
		}
	}

	users[m.Author.ID].lastSaw = time.Now()
	users[m.Author.ID].total++

	return true
}
