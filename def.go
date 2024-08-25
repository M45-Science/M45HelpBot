package main

import (
	"time"
)

const (
	maxAttempts          = 50
	throttlePerUser      = time.Minute * 2
	throttleGlobal       = time.Second * 30
	maxPerUser           = 3
	maxGlobal            = 10
	maxCombinedResponses = 3
	rebootTime           = time.Hour * 12
)

var (
	/* Discord data */
	discToken    string
	staffRole    string
	guildID      string
	staffChannel string

	lastReply              time.Time
	users                  map[string]*userData = map[string]*userData{}
	discordConnectAttempts int
	totalMsgCount          int
)

type helpData struct {
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
