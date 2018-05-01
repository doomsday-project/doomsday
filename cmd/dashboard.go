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

	within3DaysBound := time.Duration(time.Hour * 24 * 3)
	within3Days := results.Filter(doomsday.CacheItemFilter{
		Beyond: &expiredBound,
		Within: &within3DaysBound,
	})

	if len(within3Days) > 0 {
		header := tablewriter.NewWriter(os.Stdout)
		header.SetHeader([]string{"3 DAYS"})
		header.SetHeaderColor(tablewriter.Colors{
			tablewriter.Bold,
			tablewriter.BgBlackColor,
			tablewriter.FgHiYellowColor,
		})
		header.SetHeaderLine(false)
		header.Render()

		printList(within3Days)
	}

	within2WeeksBound := time.Duration(time.Hour * 24 * 7 * 2)
	within2Weeks := results.Filter(doomsday.CacheItemFilter{
		Beyond: &within3DaysBound,
		Within: &within2WeeksBound,
	})

	if len(within2Weeks) > 0 {
		header := tablewriter.NewWriter(os.Stdout)
		header.SetHeader([]string{"2 WEEKS"})
		header.SetHeaderColor(tablewriter.Colors{
			tablewriter.Bold,
			tablewriter.BgBlackColor,
			tablewriter.FgHiGreenColor,
		})
		header.SetHeaderLine(false)
		header.Render()

		printList(within2Weeks)
	}

	within4WeeksBound := time.Duration(time.Hour * 24 * 7 * 4)
	within4Weeks := results.Filter(doomsday.CacheItemFilter{
		Beyond: &within2WeeksBound,
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
