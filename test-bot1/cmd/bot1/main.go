package main

import (
	"fmt"
	"math/rand"
	"test-bot1/bot"
	"time"
)

// const (
// 	MAX_CLIENTS = 100
// )

var activeBots = 0

func main() {
	fmt.Println("Welcome to Bot1 script...")
	fmt.Println("We are going to have some F.U.N. today.")

	ticker := time.NewTicker(300 * time.Millisecond)
	ticksUntilNextSpawn := 3

	for {
		<-ticker.C
		if activeBots >= 5 {
			continue
		}
		ticksUntilNextSpawn--

		if ticksUntilNextSpawn <= 0 {
			ticksUntilNextSpawn = rand.Intn(5)
			activeBots++
			go runBot()
		}

	}
}

func runBot() {
	ticker := time.NewTicker(160 * time.Millisecond)
	b := bot.NewBot()
	botNum := activeBots
	fmt.Println("New Bot number", activeBots)
	for {
		<-ticker.C
		if !b.Active {
			b.Socket.Close()
			break
		}
		b.Move()
		fmt.Println("Move bot", botNum)
	}
}
