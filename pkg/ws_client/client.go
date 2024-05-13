package ws_client

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/log"
)

type clientServer map[string]map[string]*websocket.Conn

type Client struct {
	Conn   *websocket.Conn
	Server string
	User   string
}

type Broadcast struct {
	From Client
	Msg  string
}

var (
	clients    = make(clientServer)
	register   = make(chan Client)
	unregister = make(chan Client)
	broadcast  = make(chan Broadcast)
)

func StkConfig() (clientServer, chan Client, chan Client, chan Broadcast) {
	return clients, register, unregister, broadcast
}

func RemoveClient(c *Client) {
	if conn, ok := clients[c.Server][c.User]; ok {
		delete(clients[c.Server], c.User)
		conn.Close() // close the connection before deleting client mapped
	}
	delete(clients[c.Server], c.User)
	if len(clients[c.Server]) == 0 {
		delete(clients, c.Server)
	}
}

func SktHandler() {
	for {
		select {
		case cli := <-register:
			if _, ok := clients[cli.Server]; !ok {
				clients[cli.Server] = make(map[string]*websocket.Conn)
			}
			clients[cli.Server][cli.User] = cli.Conn
			log.Info("Client registered: ", cli.Server, cli.User)
		case cli := <-unregister:
			RemoveClient(&cli)
			log.Info("Client unregistered: ", cli.Server, cli.User)
		case msg := <-broadcast:
			for sv, users := range clients {
				if sv == msg.From.Server { // list of users in the same server
					for user, conn := range users {
						if sv != msg.From.Server || user != msg.From.User { // exclude the sender
							if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Msg)); err != nil {
								log.Error("Error broadcasting message to ", sv, user, err)
								RemoveClient(&Client{
									User:   user,
									Server: sv,
								})
								conn.WriteMessage(websocket.CloseMessage, []byte{})
								conn.Close()
							}
						}
					}
				}
			}

		}
	}
}
