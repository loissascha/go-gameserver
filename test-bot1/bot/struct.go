package bot

import (
	"fmt"
	"math/rand"
	gameserver "test-bot1/gameServer"

	"github.com/gorilla/websocket"
)

type Bot struct {
	Active bool
	PosX   float64
	PosY   float64
	Socket *websocket.Conn
}
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewBot() *Bot {
	conn, _, err := websocket.DefaultDialer.Dial(gameserver.ServerAddress, nil)
	if err != nil {
		panic(err)
	}
	b := Bot{Active: true}
	b.Socket = conn
	return &b
}

func (b *Bot) Move() {
	xpm := rand.Intn(100)
	if xpm > 50 {
		b.PosX += float64(rand.Intn(25))
	} else {
		b.PosX -= float64(rand.Intn(25))
	}

	ypm := rand.Intn(100)
	if ypm > 50 {
		b.PosY += float64(rand.Intn(25))
	} else {
		b.PosY -= float64(rand.Intn(25))
	}

	b.sendPosition()
}

func (b *Bot) sendPosition() {
	position := [2]float64{float64(b.PosX), float64(b.PosY)}
	msg := Message{
		Type: "update_position",
		Data: position,
	}
	err := b.Socket.WriteJSON(msg)
	if err != nil {
		fmt.Println("Can't send player pos update:", err)
		return
	}
}
