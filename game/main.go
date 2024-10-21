package main

import (
	"encoding/json"
	"fmt"
	"image"
	"math"

	// "math/rand"
	"os"
	"time"

	_ "image/png"

	"github.com/gopxl/pixel/v2"
	"github.com/gopxl/pixel/v2/backends/opengl"
	"github.com/gopxl/pixel/v2/ext/text"
	"github.com/gorilla/websocket"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

var (
	camPos       = pixel.ZV // = playerPos ?!
	camSpeed     = 500.0
	camZoom      = 1.0
	camZoomSpeed = 1.2
)
var gameWindowConfig opengl.WindowConfig
var gameWindow *opengl.Window
var gameServer *websocket.Conn
var serverAddress = "ws://localhost:11811/ws"

type OtherPlayerResult struct {
	ID       string     `json:"id"`
	Position [2]float64 `json:"position"`
}
type OtherPlayer struct {
	id       string
	position [2]float64
	exists   bool
}
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var otherPlayers []OtherPlayer

func run() {
	// otherPlayers = append(otherPlayers, OtherPlayer{id: "1", position: [2]float64{float64(10), float64(50)}})
	configWindow()
	spawnWindow()
	playerSprite := getImageSprite("player_self.png")
	playerOtherSprite := getImageSprite("player_other.png")

	textAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	fpsText := text.New(pixel.V(0, 0), textAtlas)

	last := time.Now()
	var (
		frames = 0
		second = time.Tick(time.Second)
	)
	for !gameWindow.Closed() {
		////// INIT //////

		// Clear window
		gameWindow.Clear(colornames.Black)
		// Calculate delta time
		dt := time.Since(last).Seconds()
		last = time.Now()

		/////////////////

		//////// UNSCALED //////////

		// Set matrix to the camera position (so camera movement works)
		cam := pixel.IM.Moved(gameWindow.Bounds().Center().Sub(camPos))
		gameWindow.SetMatrix(cam)

		// Draw the debug text
		textpos := cam.Unproject(
			gameWindow.Bounds().Center().Sub(
				pixel.V(
					gameWindow.Bounds().Max.X/2,
					(gameWindow.Bounds().Max.Y/2)*-1+20,
				)))
		fpsText.Draw(gameWindow, pixel.IM.Moved(textpos))

		////////////////////

		////////////// SCALED ///////////

		// Set matrix to the camera position (so camera movement works)
		cam = pixel.IM.Scaled(camPos, camZoom).Moved(gameWindow.Bounds().Center().Sub(camPos))
		gameWindow.SetMatrix(cam)

		// Draw Player
		mat := pixel.IM
		mat = mat.ScaledXY(pixel.ZV, pixel.V(0.1, 0.1))
		pos := cam.Unproject(gameWindow.Bounds().Center())
		mat = mat.Moved(pos)
		gameWindow.SetSmooth(true)
		playerSprite.Draw(gameWindow, mat)

		// Draw other players
		for _, p := range otherPlayers {
			mat = pixel.IM
			mat = mat.ScaledXY(pixel.ZV, pixel.V(0.1, 0.1))
			mat = mat.Moved(pixel.V(p.position[0], p.position[1]))
			playerOtherSprite.Draw(gameWindow, mat)
		}

		////////// INPUT ////////

		// plant trees
		// if win.JustPressed(pixel.MouseButtonLeft) {
		// tree := pixel.NewSprite(treesheet, treesFrames[rand.Intn(len(treesFrames))])
		// mouse := cam.Unproject(win.MousePosition())
		// tree.Draw(treebatch, pixel.IM.Scaled(pixel.ZV, 4).Moved(mouse))
		// }

		// Player movement
		hasPosUpdate := false
		if gameWindow.Pressed(pixel.KeyS) {
			camPos.X -= camSpeed * dt
			hasPosUpdate = true
		}
		if gameWindow.Pressed(pixel.KeyF) {
			camPos.X += camSpeed * dt
			hasPosUpdate = true
		}
		if gameWindow.Pressed(pixel.KeyD) {
			camPos.Y -= camSpeed * dt
			hasPosUpdate = true
		}
		if gameWindow.Pressed(pixel.KeyE) {
			camPos.Y += camSpeed * dt
			hasPosUpdate = true
		}
		if hasPosUpdate {
			sendPosUpdate()
		}
		camZoom *= math.Pow(camZoomSpeed, gameWindow.MouseScroll().Y)
		if camZoom < 0.5 {
			camZoom = 0.5
		} else if camZoom > 3 {
			camZoom = 3
		}

		gameWindow.Update()

		/////////// AFTER UPDATE ///////////

		// FPS counter updates
		frames++
		select {
		case <-second:
			// win.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			fpsText.Clear()
			fpsText.Color = colornames.Red
			fmt.Fprintf(fpsText, "FPS: %d", frames)
			frames = 0
		default:
		}

		//////////////////////////////////
	}
}

func sendPosUpdate() {
	position := [2]float64{float64(camPos.X), float64(camPos.Y)}
	msg := Message{
		Type: "update_position",
		Data: position,
	}
	err := gameServer.WriteJSON(msg)
	if err != nil {
		fmt.Println("Can't send player pos update:", err)
		return
	}
}

func parseData(data interface{}, v interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}

func getPosUpates() {
	for {
		var res Message
		err := gameServer.ReadJSON(&res)
		if err != nil {
			fmt.Println("Read error:", err)
			continue
		}

		switch res.Type {
		case "players_data":
			var playersData []OtherPlayerResult
			err := parseData(res.Data, &playersData)
			if err != nil {
				fmt.Println("Can't read player update")
				continue
			}
			// set all players to non exist
			for i := range otherPlayers {
				otherPlayers[i].exists = false
			}

			for _, v := range playersData {
				playerExists := false
				for i, p := range otherPlayers {
					if p.id == v.ID {
						playerExists = true
						otherPlayers[i].position = v.Position
						otherPlayers[i].exists = true
					}
				}
				if !playerExists {
					otherPlayers = append(otherPlayers, OtherPlayer{id: v.ID, position: v.Position, exists: true})
				}
			}

			// only leave those who exist
			newOtherPlayers := []OtherPlayer{}
			for _, p := range otherPlayers {
				if p.exists {
					newOtherPlayers = append(newOtherPlayers, p)
				}
			}
			otherPlayers = newOtherPlayers

			fmt.Println("Received players data:", playersData)
			break
		}

	}
}

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func configWindow() {
	gameWindowConfig = opengl.WindowConfig{
		Title:  "Multiplayer Pixel Game Test",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
}

func spawnWindow() {
	win, err := opengl.NewWindow(gameWindowConfig)
	if err != nil {
		panic(err)
	}
	gameWindow = win
}

func getImageSprite(path string) *pixel.Sprite {
	picture, err := loadPicture(path)
	if err != nil {
		panic(err)
	}
	sprite := pixel.NewSprite(picture, picture.Bounds())
	return sprite
}

func main() {
	conn, _, err := websocket.DefaultDialer.Dial(serverAddress, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	gameServer = conn

	sendPosUpdate()
	go getPosUpates()

	opengl.Run(run)
}
