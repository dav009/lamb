package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jroimartin/gocui"
)

var (
	svc                *lambda.Lambda
	cw                 *cloudwatchlogs.CloudWatchLogs
	functionNameFilter string
)

func ConnectToLambdaAPI() *lambda.Lambda {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := lambda.New(sess)
	return svc
}

func ConnectToCloudwatchAPI() *cloudwatchlogs.CloudWatchLogs {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := cloudwatchlogs.New(sess)
	return svc
}

func getLogs(logroup string) ([]*cloudwatchlogs.OutputLogEvent, error) {
	allEvents := []*cloudwatchlogs.OutputLogEvent{}
	streamInput := &cloudwatchlogs.DescribeLogStreamsInput{
		OrderBy:      aws.String("LastEventTime"),
		LogGroupName: aws.String(logroup),
		Limit:        aws.Int64(2),
		Descending:   aws.Bool(true),
	}
	streams, err := cw.DescribeLogStreams(streamInput)
	if err != nil {
		return nil, err
	}
	for _, stream := range streams.LogStreams {
		input := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(logroup),
			LogStreamName: stream.LogStreamName}
		r, err := cw.GetLogEvents(input)
		if err != nil {
			return nil, err
		}
		allEvents = append(r.Events, allEvents...)

	}
	return allEvents, nil
}

func getLogGroup(functionName string) ([]*cloudwatchlogs.LogGroup, error) {
	cloudWatchStream := fmt.Sprintf("/aws/lambda/%s", functionName)
	logGroupsInput := &cloudwatchlogs.DescribeLogGroupsInput{LogGroupNamePrefix: aws.String(cloudWatchStream)}

	logGroups, err := cw.DescribeLogGroups(logGroupsInput)
	if err != nil {
		return nil, err
	}
	return logGroups.LogGroups, nil
}

func getLambdaStatus(functionName string) (*lambda.FunctionConfiguration, error) {
	input := &lambda.GetFunctionConfigurationInput{
		FunctionName: aws.String(functionName),
	}
	status, _ := svc.GetFunctionConfiguration(input)
	return status, nil
}

func ListLambdaFunctions() ([]*lambda.FunctionConfiguration, error) {

	// Create Lambda service client

	result, err := svc.ListFunctions(nil)

	if err != nil {
		return nil, err
	}

	return result.Functions, nil
}

func updateLambdaStatusView(g *gocui.Gui, v *gocui.View, functionName string) error {
	v, _ = g.View("state")

	lambdaConfig, err := getLambdaStatus(functionName)
	if err != nil {
		return err
	}
	v.Clear()

	fields := map[string]string{
		"Description":   aws.StringValue(lambdaConfig.Description),
		"Last modified": aws.StringValue(lambdaConfig.LastModified),
		"memory size":   strconv.Itoa(int(aws.Int64Value(lambdaConfig.MemorySize))),
		"role":          aws.StringValue(lambdaConfig.Role),
		"runtime":       aws.StringValue(lambdaConfig.Runtime),
		"timeout":       strconv.Itoa(int(aws.Int64Value(lambdaConfig.Timeout)))}

	keys := make([]string, 0)
	for k, _ := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, title := range keys {
		value := fields[title]
		fmt.Fprintf(v, "\033[32;5m%s\033[0m %s\n", title, value)
	}
	return nil
}

func colorizeLogLine(logLine string, timestamp int64) string {
	tm := time.Unix(timestamp, 0)
	if strings.HasPrefix(logLine, "START RequestId:") || strings.HasPrefix(logLine, "END RequestId:") || strings.HasPrefix(logLine, "REPORT RequestId:") {

		return fmt.Sprintf("\033[32;5m%s\033[0m - \033[32;7m%s\033[0m\n", tm, logLine)

	} else {
		return fmt.Sprintf("\033[32;5m%s\033[0m - %s\n", tm, logLine)

	}
}

func updateLogsView(g *gocui.Gui, v *gocui.View, functionName string) error {
	v, _ = g.View("logs")
	v.Clear()
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)
	v.Autoscroll = true
	logGroups, err := getLogGroup(functionName)
	if err != nil {
		return err
	}
	for _, logGroup := range logGroups {

		logs, err := getLogs(aws.StringValue(logGroup.LogGroupName))
		if err != nil {
			return err
		}

		for _, log := range logs {
			logLine := aws.StringValue(log.Message)
			fmt.Fprintf(v, colorizeLogLine(logLine, aws.Int64Value(log.Timestamp)/1000))
		}

	}
	return nil
}

func updateView(g *gocui.Gui, v *gocui.View) error {
	var functionName string
	var err error

	v, err = g.View("side")
	if err != nil {
		return err
	}

	_, cy := v.Cursor()
	if functionName, err = v.Line(cy); err != nil {
		functionName = ""
	}

	if functionName != "" {

		err = updateLambdaStatusView(g, v, functionName)
		if err != nil {
			return err
		}

		err = updateLogsView(g, v, functionName)
		if err != nil {
			return err
		}
	}

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("side", -1, -1, 30, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		functions, err := ListLambdaFunctions()
		if err != nil {
			return err
		}

		functionNames := []string{}
		for _, f := range functions {
			name := aws.StringValue(f.FunctionName)

			if functionNameFilter != "" {

				if strings.Contains(name, functionNameFilter) {
					functionNames = append(functionNames, name)
				}

			} else {
				functionNames = append(functionNames, name)
			}

		}
		sort.Strings(functionNames)

		for _, n := range functionNames {
			fmt.Fprintln(v, n)
		}

	}

	if v, err := g.SetView("logs", 30, -1, maxX, maxY-11); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Wrap = true
		v.Autoscroll = true
		v.Frame = true
		v.Highlight = true
		v.SelBgColor = gocui.ColorBlue
		v.SelFgColor = gocui.ColorBlack
	}

	if v, err := g.SetView("state", 30, maxY-10, maxX, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.Frame = true
		v.Wrap = true
		if _, err := g.SetCurrentView("side"); err != nil {
			return err
		}

	}

	return nil
}

func main() {

	if len(os.Args) > 1 {
		functionNameFilter = os.Args[1]
	}
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true
	svc = ConnectToLambdaAPI()
	cw = ConnectToCloudwatchAPI()

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("logs", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		return err
	}

	if err := g.SetKeybinding("logs", gocui.KeyEnter, gocui.ModNone, updateView); err != nil {
		return err
	}
	if err := g.SetKeybinding("logs", gocui.KeyArrowDown, gocui.ModNone, logsDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("logs", gocui.KeyArrowUp, gocui.ModNone, logsUp); err != nil {
		return err
	}

	if err := g.SetKeybinding("side", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("side", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding("side", gocui.KeyEnter, gocui.ModNone, updateView); err != nil {
		return err
	}

	return nil
}

/*ui */

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	current := g.CurrentView()
	if v == nil || current.Name() == "logs" {
		_, err := g.SetCurrentView("side")
		return err
	}
	_, err := g.SetCurrentView("logs")
	return err
}

func logsDown(g *gocui.Gui, v *gocui.View) error {
	scroll(g, 5)
	return nil
}

func logsUp(g *gocui.Gui, v *gocui.View) error {
	scroll(g, -5)
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func scroll(g *gocui.Gui, dy int) {
	// Grab the view that we want to scroll.
	v, _ := g.View("logs")

	// Get the size and position of the view.
	_, y := v.Size()
	_, oy := v.Origin()

	// If we're at the bottom...
	if oy+dy > strings.Count(v.ViewBuffer(), "\n")-y-1 {
		// Set autoscroll to normal again.
		v.Autoscroll = true
	} else {
		// Set autoscroll to false and scroll.
		v.Autoscroll = false
		v.MoveCursor(0, dy, false)
	}
}
