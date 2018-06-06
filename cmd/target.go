package main

import (
	"fmt"
	"net/url"
)

type targetCmd struct {
	Name       *string
	Address    *string
	SkipVerify *bool
}

func (t *targetCmd) Run() error {
	if t.Name == nil || *t.Name == "" {
		if target == nil {
			fmt.Println("No backend currently targeted")
		} else {
			fmt.Println(target)
		}
		return nil
	}

	if t.Address != nil && *t.Address != "" {
		addrUrl, err := url.Parse(*t.Address)
		if err != nil {
			return fmt.Errorf("Could not parse given address as URL")
		}

		if addrUrl.Port() == "" {
			addrUrl.Host = fmt.Sprintf("%s:8111", addrUrl.Host)
		}

		if addrUrl.Scheme != "http" && addrUrl.Scheme != "https" {
			return fmt.Errorf("Address contains unsupported protocol `%s'", addrUrl.Scheme)
		}

		err = cliConf.Add(CLITarget{
			Name:       *t.Name,
			Address:    addrUrl.String(),
			SkipVerify: *t.SkipVerify,
		})
		if err != nil {
			return err
		}
	}

	err := cliConf.SetCurrent(*t.Name)
	if err != nil {
		return err
	}

	fmt.Println(target)
	return nil
}
