package main

import "fmt"

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
		err := cliConf.Add(CLITarget{
			Name:       *t.Name,
			Address:    *t.Address,
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
