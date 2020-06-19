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
	header.SetHeader([]string{"WORKERS"})
	header.SetHeaderLine(false)
	header.Render()

	printWorkerList(state.Workers)

	header = tablewriter.NewWriter(os.Stdout)
	header.SetHeader([]string{"RUNNING"})
	header.SetHeaderLine(false)
	header.Render()

	printSchedTaskList(state.Running, true)

	header = tablewriter.NewWriter(os.Stdout)
	header.SetHeader([]string{"PENDING"})
	header.SetHeaderLine(false)
	header.Render()

	printSchedTaskList(state.Pending, false)
	return nil
}

func printWorkerList(workers []doomsday.GetSchedulerWorker) {
	fmt.Printf("\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetReflowDuringAutoWrap(false)
	table.SetHeader([]string{"ID", "State", "For"})
	table.SetAlignment(tablewriter.ALIGN_RIGHT)

	now := time.Now()
	for _, worker := range workers {
		timeSinceStr := now.Sub(time.Unix(worker.StateAt, 0)).Truncate(100 * time.Millisecond).String()
		table.Append([]string{strconv.FormatUint(uint64(worker.ID), 10), worker.State, timeSinceStr})
	}
	table.Render()
	fmt.Printf("\n")
}

func printSchedTaskList(tasks []doomsday.GetSchedulerTask, showWorker bool) {
	fmt.Printf("\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetReflowDuringAutoWrap(false)
	headers := []string{"ID", "At", "Backend", "Kind", "Reason", "State"}
	if showWorker {
		headers = append(headers, "Worker")
	}
	table.SetHeader(headers)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)

	now := time.Now()
	for _, task := range tasks {
		timeUntilStr := time.Unix(task.At, 0).Sub(now).Truncate(100 * time.Millisecond).String()
		values := []string{
			strconv.FormatUint(uint64(task.ID), 10),
			timeUntilStr,
			task.Backend,
			task.Kind,
			task.Reason,
			task.State,
		}

		if showWorker {
			values = append(values, strconv.FormatInt(int64(task.WorkerID), 10))
		}

		table.Append(values)
	}
	table.Render()
	fmt.Printf("\n")
}
