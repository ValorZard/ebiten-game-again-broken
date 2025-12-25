// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// data-channels-detach is an example that shows how you can detach a data channel.
// This allows direct access the underlying [pion/datachannel](https://github.com/pion/datachannel).
// This allows you to interact with the data channel using a more idiomatic API based on
// the `io.ReadWriteCloser` interface.
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"valorzard/ebiten-again/common"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/pion/webrtc/v4"
)

var peerConnection *webrtc.PeerConnection

const url = "http://localhost:8080"

func doWhep() {
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}
	<-gatherComplete

	whepUrl := url + "/whep"
	sdpOffer := []byte(peerConnection.LocalDescription().SDP)

	req, err := http.NewRequest(http.MethodPost, whepUrl, bytes.NewReader(sdpOffer))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/sdp")
	req.Header.Set("Authorization", `Bearer none`)

	res, getErr := http.DefaultClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}
	defer res.Body.Close()

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(body),
	}
	if err = peerConnection.SetRemoteDescription(answer); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Finished WHEP setup!")
}

func doWhip() {
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}
	<-gatherComplete

	whipUrl := url + "/whip"
	sdpOffer := []byte(peerConnection.LocalDescription().SDP)

	req, err := http.NewRequest(http.MethodPost, whipUrl, bytes.NewReader(sdpOffer))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/sdp")
	req.Header.Set("Authorization", `Bearer none`)

	res, getErr := http.DefaultClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}
	defer res.Body.Close()

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(body),
	}
	if err = peerConnection.SetRemoteDescription(answer); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Finished WHIP setup!")
}

type Game struct {
	debugui     debugui.DebugUI
	textLog     string
	textMessage string
	dataChannel *webrtc.DataChannel
	playerX     float64
	playerY     float64
	username    string
	playerList  PlayerList
	tick        uint16
	// we need a way to store our own last sent packet for prediction
	currentPacket common.NetPacket
}

type PlayerList struct {
	mutex   sync.RWMutex
	Players map[string]GameState
}

type GameState struct {
	PositionX float64
	PositionY float64
}

var img *ebiten.Image

func init() {
	var err error
	img, _, err = ebitenutil.NewImageFromFile("test.png")
	if err != nil {
		log.Fatal(err)
	}

	// Register NetPacket type for gob encoding/decoding
	gob.Register(common.NetPacket{})

}

func createGame() *Game {
	return &Game{
		playerX: 50,
		playerY: 50,
		playerList: PlayerList{
			Players: make(map[string]GameState),
		},
	}
}

func (g *Game) Update() error {
	// update internal tick
	g.tick = (g.tick + 1) % common.MAX_NET_PACKET_TICK

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.playerY--
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.playerY++
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.playerX--
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.playerX++
	}

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Debugui Window", image.Rect(0, 0, 320, 400), func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1}, []int{-1, 20, 20})
			// Place all your widgets inside a ctx.Window's callback.
			// Specify a presssing-button event handler by On.
			ctx.Panel(func(layout debugui.ContainerLayout) {
				ctx.Text(g.textLog)
			})
			ctx.Button("WHIP").On(func() {
				doWhip()
			})

			ctx.Button("WHEP").On(func() {
				doWhep()
			})
		})
		return nil
	}); err != nil {
		return err
	}
	// Create an encoder and send a value.
	var packet bytes.Buffer // Stand-in for the network.
	enc := gob.NewEncoder(&packet)
	err := enc.Encode(common.NetPacket{
		Username:  g.username,
		PositionX: g.playerX,
		PositionY: g.playerY,
	})
	if err != nil {
		return err
	}
	g.dataChannel.Send(packet.Bytes())
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.playerX, g.playerY)
	screen.DrawImage(img, op)
	// draw other players
	g.playerList.mutex.RLock()
	defer g.playerList.mutex.RUnlock()
	for _, player := range g.playerList.Players {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(player.PositionX, player.PositionY)
		screen.DrawImage(img, op)
	}
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	// create game object
	game := createGame()
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection using the default API object
	// Don't detach data channels since we won't be able to receive them through OnDataChannel
	err := error(nil)
	peerConnection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			game.textLog += fmt.Sprintf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		game.textLog += fmt.Sprintf("Peer Connection State has changed: %s\n", state.String())

		if state == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure.
			// It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			game.textLog += fmt.Sprintln("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}

		if state == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			game.textLog += fmt.Sprintln("Peer Connection has gone to closed exiting")
			os.Exit(0)
		}
	})

	/*
		ordered := false
		maxRetransmits := uint16(0)

			dataChannelInit := &webrtc.DataChannelInit{
				Ordered:        &ordered,
				MaxRetransmits: &maxRetransmits,
			}*/
	dataChannel, err := peerConnection.CreateDataChannel("foo", nil)
	if err != nil {
		panic(err)
	}

	dataChannel.OnOpen(func() {
		game.textLog += fmt.Sprintf("Data channel '%s'-'%d' open.\n", dataChannel.Label(), dataChannel.ID())
	})
	dataChannel.OnClose(func() {
		game.textLog += fmt.Sprintf("Data channel '%s'-'%d' closed.\n", dataChannel.Label(), dataChannel.ID())
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// move this to a goroutine if doing heavy processing
		go func() {
			var messageBytes bytes.Buffer
			dec := gob.NewDecoder(&messageBytes)
			messageBytes.Write(msg.Data)
			var packet common.NetPacket
			if err := dec.Decode(&packet); err != nil {
				game.textLog += fmt.Sprintf("Failed to decode incoming message: %v\n", err)
				return
			}
			// this welcome packet gives us our assigned username
			if game.username == "" {
				game.username = packet.Username
				game.textLog += fmt.Sprintf("Assigned username: %s\n", game.username)
			} else if packet.Username == game.username {
				// update our own position
				if packet.Tick <= game.currentPacket.Tick {
					// outdated packet, ignore
					return
				}
				// use a shitty version of prediction by interpolating positions
				game.currentPacket = packet
				game.playerX = (game.currentPacket.PositionX + game.playerX) / 2
				game.playerY = (game.currentPacket.PositionY + game.playerY) / 2
			} else {
				// update player position
				game.playerList.mutex.Lock()
				defer game.playerList.mutex.Unlock()
				game.playerList.Players[packet.Username] = GameState{
					PositionX: packet.PositionX,
					PositionY: packet.PositionY,
				}
			}
		}()
	})

	game.dataChannel = dataChannel

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		game.textLog += fmt.Sprintf("New DataChannel %s %d\n", dc.Label(), dc.ID())
	})

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
