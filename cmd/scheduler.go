package main

import (
	"fmt"
	"os"
	"time"

	"github.com/doomsday-project/doomsday/client/doomsday"
	"github.com/olekukonko/tablewriter"
	"github.com/starkandwayne/goutils/ansi"
)

type schedulerCmd struct{}

func (*schedulerCmd) Run() error {
	state, err := client.GetSchedulerState()
	if err != nil {
		return err
	}

	printSchedTaskList(state.Tasks)
	return nil
}

func printSchedTaskList(tasks []doomsday.GetSchedulerTask) {
	fmt.Printf("\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetReflowDuringAutoWrap(false)
	table.SetHeader([]string{"At", "Reason", "Kind", "Ready"})

	readyStr := ansi.Sprintf("@G{YES}")
	notReadyStr := ansi.Sprintf("@R{NO}")
	now := time.Now()
	for _, task := range tasks {
		timeUntilStr := time.Unix(task.At, 0).Sub(now).String()
		readyOutStr := notReadyStr
		if task.Ready {
			readyOutStr = readyStr
		}
		table.Append([]string{
			timeUntilStr,
			task.Reason,
			task.Kind,
			readyOutStr,
		})
	}
	table.Render()
	fmt.Printf("\n")
}
