package main

import (
	"flag"
	"log"
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/textures1245/go-fiber-socket/pkg/ws_client"
)

func main() {
	app := fiber.New()
	app.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return strings.Contains(c.Route().Path, "/ws")
		},
	}))

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)

			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	go ws_client.SktHandler()

	app.Get("/ws/:server_id/:user_id", websocket.New(func(c *websocket.Conn) {
		// c.Locals is added to the *websocket.Conn
		log.Println(c.Locals("allowed"))  // true
		log.Println(c.Cookies("session")) // ""

		client := &ws_client.Client{
			Conn:   c,
			Server: c.Params("server_id"),
			User:   c.Params("user_id"),
		}
		_, reg, unreg, brd := ws_client.StkConfig()

		defer func() {
			unreg <- *client // unregister all clients
			c.Close()
		}()

		reg <- *client // register the client

		for {
			msgType, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Panicf("error: %v", err)
				}
				return
			}

			if msgType == websocket.TextMessage {
				brd <- ws_client.Broadcast{
					From: *client,
					Msg:  string(msg),
				}

			}
		}

	}))

	addr := flag.String("addr", ":3000", "http service address")
	flag.Parse()
	app.Listen(*addr)
}
