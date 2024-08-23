package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	maxAttempts          = 50
	throttlePerUser      = time.Minute * 2
	throttleGlobal       = time.Second * 30
	maxPerUser           = 5
	maxGlobal            = 12
	maxCombinedResponses = 3
	rebootTime           = time.Hour * 24
)

var (
	/* Discord data */
	ds        *discordgo.Session
	discToken string
	staffRole string
	guildID   string

	lastReply              time.Time
	users                  map[string]*userData = map[string]*userData{}
	discordConnectAttempts int
	totalMsgCount          int
)

type helpData struct {
	Priority   int      `json:",omitempty"`
	Wildcards  []string `json:",omitempty"`
	Words      []string `json:",omitempty"`
	Exclude    []string `json:",omitempty"`
	ReplyLines []string `json:",omitempty"`
}

type userData struct {
	id      string
	lastSaw time.Time
	total   int
}
