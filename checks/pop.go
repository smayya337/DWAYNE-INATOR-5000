package checks

import (
	"fmt"
	"github.com/knadh/go-pop3"
)

type Pop struct {
	checkBase
}

func (c Pop) Run(teamID uint, boxIp string, res chan Result) {
	username, password := getCreds(teamID, c.CredLists, c.Name)

	// Initialize the client.
	p := pop3.New(pop3.Opt{
		Host: boxIp,
		Port: c.Port,
		TLSEnabled: false,
	})

	// Create a new connection. POP3 connections are stateful and should end
	// with a Quit() once the opreations are done.
	conn, err := p.NewConn()
	if err != nil {
		res <- Result{
			Error: "connecting to POP3 server failed",
			Debug: "error: " + err.Error() + ", creds " + username + ":" + password,
		}
		return
	}
	defer conn.Quit()

	// Authenticate.
	if err := conn.Auth(username, password); err != nil {
		res <- Result{
			Error: "authenticating with POP3 server failed",
			Debug: "error: " + err.Error() + ", creds " + username + ":" + password,
		}
		return
	}

	// Print the total number of messages and their size.
	count, size, _ := conn.Stat()
	fmt.Println("total messages=", count, "size=", size)

	// Pull the list of all message IDs and their sizes.
	msgs, _ := conn.List(0)
	for _, m := range msgs {
		fmt.Println("id=", m.ID, "size=", m.Size)
	}

	// Pull all messages on the server. Message IDs go from 1 to N.
	for id := 1; id <= count; id++ {
		m, _ := conn.Retr(id)

		fmt.Println(id, "=", m.Header.Get("subject"))

		// To read the multi-part e-mail bodies, see:
		// https://github.com/emersion/go-message/blob/master/example_test.go#L12
	}

	// Delete all the messages. Server only executes deletions after a successful Quit()
	for id := 1; id <= count; id++ {
		conn.Dele(id)
	}

	res <- Result{
		Status: true,
		Debug:  "creds used were " + username + ":" + password,
	}
}