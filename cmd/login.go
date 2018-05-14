package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

type loginCmd struct {
	Username *string
	Password *string
}

func (l *loginCmd) Run() error {
	if l.Username == nil || *l.Username == "" {
		fmt.Printf("Username: ")
		_, err := fmt.Scanln(l.Username)
		if err != nil {
			return err
		}
	}

	if l.Password == nil || *l.Password == "" {
		fmt.Printf("Password: ")
		tmpPass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println("")
		if err != nil {
			return err
		}

		tmpPassStr := string(tmpPass)
		l.Password = &tmpPassStr
	}

	err := client.UserpassAuth(*l.Username, *l.Password)
	if err != nil {
		return fmt.Errorf("Could not authenticate: %s", err)
	}

	toAdd := *cliConf.CurrentTarget()
	toAdd.Token = client.Token
	cliConf.Add(toAdd)
	return nil
}
