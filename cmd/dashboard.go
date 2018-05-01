package main

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/thomasmmitchell/doomsday"
)

type dashboardCmd struct {
}

func (d *dashboardCmd) Run() error {
	results, err := client.GetCache()
	if err != nil {
		return err
	}

	expiredBound := time.Duration(0)
	expired := results.Filter(doomsday.CacheItemFilter{
		Within: &expiredBound,
	})

	if len(expired) > 0 {
		header := tablewriter.NewWriter(os.Stdout)
		header.SetHeader([]string{"EXPIRED"})
		header.SetHeaderColor(tablewriter.Colors{
			tablewriter.Bold,
			tablewriter.BgBlackColor,
			tablewriter.FgHiRedColor,
		})
		header.SetHeaderLine(false)
		header.Render()

		t := tablewriter.NewWriter(os.Stdout)
		t.SetHeader([]string{"Common Name", "Path"})
		for _, v := range expired {
			t.Append([]string{v.CommonName, v.Path})
		}
		t.SetBorder(false)
		t.SetRowLine(true)
		t.Render()
	}

	within24HoursBound := time.Duration(time.Hour * 24)
	within24Hours := results.Filter(doomsday.CacheItemFilter{
		Beyond: &expiredBound,
		Within: &within24HoursBound,
	})

	if len(within24Hours) > 0 {
		header := tablewriter.NewWriter(os.Stdout)
		header.SetHeader([]string{"24 HOURS"})
		header.SetHeaderColor(tablewriter.Colors{
			tablewriter.Bold,
			tablewriter.BgBlackColor,
			tablewriter.FgHiYellowColor,
		})
		header.SetHeaderLine(false)
		header.Render()

		printList(within24Hours)
	}

	within1WeekBound := time.Duration(time.Hour * 24 * 7)
	within1Week := results.Filter(doomsday.CacheItemFilter{
		Beyond: &within24HoursBound,
		Within: &within1WeekBound,
	})

	if len(within24Hours) > 0 {
		header := tablewriter.NewWriter(os.Stdout)
		header.SetHeader([]string{"1 WEEK"})
		header.SetHeaderColor(tablewriter.Colors{
			tablewriter.Bold,
			tablewriter.BgBlackColor,
			tablewriter.FgHiGreenColor,
		})
		header.SetHeaderLine(false)
		header.Render()

		printList(within1Week)
	}

	within4WeeksBound := time.Duration(time.Hour * 24 * 7 * 4)
	within4Weeks := results.Filter(doomsday.CacheItemFilter{
		Beyond: &within1WeekBound,
		Within: &within4WeeksBound,
	})

	if len(within4Weeks) > 0 {
		header := tablewriter.NewWriter(os.Stdout)
		header.SetHeader([]string{"4 WEEKS"})
		header.SetHeaderColor(tablewriter.Colors{
			tablewriter.Bold,
			tablewriter.BgBlackColor,
			tablewriter.FgHiBlueColor,
		})
		header.SetHeaderLine(false)
		header.Render()

		printList(within4Weeks)
	}

	withinDash := results.Filter(doomsday.CacheItemFilter{
		Within: &within4WeeksBound,
	})

	if len(withinDash) == 0 {
		fmt.Println("Could not find any certs which expire soon")
	}

	return nil
}
