package main

import (
	"fmt"
	"os"

	"github.com/doomsday-project/doomsday/server/auth"
	"golang.org/x/crypto/ssh/terminal"
)

type loginCmd struct {
	Username *string
	Password *string
}

func (l *loginCmd) Run() error {
	info, err := client.Info()
	if err != nil {
		return err
	}
	var token string
	switch info.AuthType {
	case auth.AuthNop:
		token, err = l.handleNop()
	case auth.AuthUserpass:
		token, err = l.handleUserpass()
	default:
		err = fmt.Errorf("Unknown auth method: %s", info.AuthType)
	}
	if err != nil {
		return err
	}

	toAdd := *cliConf.CurrentTarget()
	toAdd.Token = token
	cliConf.Add(toAdd)

	fmt.Println("")
	fmt.Printf("Successfully authenticated to `%s'\n", toAdd.Name)
	return nil
}

func (l *loginCmd) handleNop() (string, error) {
	return "", fmt.Errorf("This doomsday server does not use authentication")
}

func (l *loginCmd) handleUserpass() (string, error) {
	if l.Username == nil || *l.Username == "" {
		fmt.Printf("Username: ")
		_, err := fmt.Scanln(l.Username)
		if err != nil {
			return "", err
		}
	}

	if l.Password == nil || *l.Password == "" {
		fmt.Printf("Password: ")
		tmpPass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println("")
		if err != nil {
			return "", err
		}

		tmpPassStr := string(tmpPass)
		l.Password = &tmpPassStr
	}

	err := client.UserpassAuth(*l.Username, *l.Password)
	if err != nil {
		return "", fmt.Errorf("Could not authenticate: %s", err)
	}

	return client.Token, nil
}
