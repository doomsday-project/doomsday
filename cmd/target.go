package main

import (
	"fmt"
	"net/url"
)

type targetCmd struct {
	Name       *string
	Address    *string
	SkipVerify *bool
	Delete     *bool
}

func (t *targetCmd) Run() error {
	var err error
	switch {
	case t.Delete != nil && *t.Delete:
		err = t.deleteTarget()
	case t.Name == nil || *t.Name == "":
		err = t.showTarget(cliConf.Current)
	case t.Address != nil && *t.Address != "":
		err = t.createTarget()
	case t.Name != nil && *t.Name != "":
		err = t.setTarget()
	}

	return err
}

func (t *targetCmd) setTarget() error {
	fmt.Printf("Setting target... ")
	target := cliConf.Find(*t.Name)
	if target == nil {
		return fmt.Errorf("No target with name `%s' exists", *t.Name)
	}

	cliConf.Current = *t.Name
	fmt.Println("Successfully set target")
	fmt.Println(target)
	return nil
}

func (t *targetCmd) showTarget(name string) error {
	if name == "" {
		fmt.Println("No backend currently targeted")
	} else {
		target := cliConf.Find(name)
		if target == nil {
			return fmt.Errorf("No backend with the name `%s' exists", name)
		}

		fmt.Println(target)
	}

	return nil
}

func (t *targetCmd) createTarget() error {
	fmt.Printf("Creating target... ")
	addrURL, err := url.Parse(*t.Address)
	if err != nil {
		return fmt.Errorf("Could not parse given address as URL")
	}

	if addrURL.Port() == "" {
		addrURL.Host = fmt.Sprintf("%s:8111", addrURL.Host)
	}

	if addrURL.Scheme != "http" && addrURL.Scheme != "https" {
		return fmt.Errorf("Address contains unsupported protocol `%s'", addrURL.Scheme)
	}

	err = cliConf.Add(CLITarget{
		Name:       *t.Name,
		Address:    addrURL.String(),
		SkipVerify: *t.SkipVerify,
	})
	if err != nil {
		return err
	}

	cliConf.SetCurrent(*t.Name)
	fmt.Printf("Successfully created target\n")
	fmt.Println(cliConf.Find(cliConf.Current))

	return nil
}

func (t *targetCmd) deleteTarget() error {
	if t.Name == nil || *t.Name == "" {
		return fmt.Errorf("If --delete flag is given, must provide a target name")
	}

	if t.Address != nil && *t.Address != "" {
		return fmt.Errorf("If --delete flag is given, only one argument to target can be given")
	}

	cliConf.Delete(*t.Name)
	if cliConf.Current == *t.Name {
		cliConf.Current = ""
	}
	return nil
}
