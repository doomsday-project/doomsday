package main

type refreshCmd struct{}

func (r *refreshCmd) Run() error {
	return client.RefreshCache()
}
