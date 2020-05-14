package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/doomsday-project/doomsday/client/doomsday"
	"github.com/olekukonko/tablewriter"
)

type schedulerCmd struct{}

func (*schedulerCmd) Run() error {
	state, err := client.GetSchedulerState()
	if err != nil {
		return err
	}

	header := tablewriter.NewWriter(os.Stdout)
	header.SetHeader([]string{"RUNNING"})
	header.SetHeaderLine(false)
	header.Render()

	printSchedTaskList(state.Running)

	header = tablewriter.NewWriter(os.Stdout)
	header.SetHeader([]string{"PENDING"})
	header.SetHeaderLine(false)
	header.Render()

	printSchedTaskList(state.Pending)
	return nil
}

func printSchedTaskList(tasks []doomsday.GetSchedulerTask) {
	fmt.Printf("\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetReflowDuringAutoWrap(false)
	table.SetHeader([]string{"ID", "At", "Backend", "Kind", "Reason", "State"})
	table.SetAlignment(tablewriter.ALIGN_RIGHT)

	now := time.Now()
	for _, task := range tasks {
		timeUntilStr := time.Unix(task.At, 0).Sub(now).Truncate(100 * time.Millisecond).String()
		table.Append([]string{
			strconv.FormatUint(uint64(task.ID), 10),
			timeUntilStr,
			task.Backend,
			task.Kind,
			task.Reason,
			task.State,
		})
	}
	table.Render()
	fmt.Printf("\n")
}
