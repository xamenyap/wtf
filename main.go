package main

// Generators
// To generate the skeleton for a new TextWidget use 'WTF_WIDGET_NAME=MySuperAwesomeWidget go generate -run=text
//go:generate -command text go run generator/textwidget.go
//go:generate text

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell"
	"github.com/olebedev/config"
	"github.com/pkg/profile"
	"github.com/radovskyb/watcher"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/bargraph"
	"github.com/wtfutil/wtf/cfg"
	"github.com/wtfutil/wtf/flags"
	"github.com/wtfutil/wtf/logger"
	"github.com/wtfutil/wtf/modules/bamboohr"
	"github.com/wtfutil/wtf/modules/circleci"
	"github.com/wtfutil/wtf/modules/clocks"
	"github.com/wtfutil/wtf/modules/cmdrunner"
	"github.com/wtfutil/wtf/modules/cryptoexchanges/bittrex"
	"github.com/wtfutil/wtf/modules/cryptoexchanges/blockfolio"
	"github.com/wtfutil/wtf/modules/cryptoexchanges/cryptolive"
	"github.com/wtfutil/wtf/modules/datadog"
	"github.com/wtfutil/wtf/modules/gcal"
	"github.com/wtfutil/wtf/modules/gerrit"
	"github.com/wtfutil/wtf/modules/git"
	"github.com/wtfutil/wtf/modules/github"
	"github.com/wtfutil/wtf/modules/gitlab"
	"github.com/wtfutil/wtf/modules/gitter"
	"github.com/wtfutil/wtf/modules/gspreadsheets"
	"github.com/wtfutil/wtf/modules/hackernews"
	"github.com/wtfutil/wtf/modules/ipaddresses/ipapi"
	"github.com/wtfutil/wtf/modules/ipaddresses/ipinfo"
	"github.com/wtfutil/wtf/modules/jenkins"
	"github.com/wtfutil/wtf/modules/jira"
	"github.com/wtfutil/wtf/modules/mercurial"
	"github.com/wtfutil/wtf/modules/nbascore"
	"github.com/wtfutil/wtf/modules/newrelic"
	"github.com/wtfutil/wtf/modules/opsgenie"
	"github.com/wtfutil/wtf/modules/pagerduty"
	"github.com/wtfutil/wtf/modules/power"
	"github.com/wtfutil/wtf/modules/resourceusage"
	"github.com/wtfutil/wtf/modules/rollbar"
	"github.com/wtfutil/wtf/modules/security"
	"github.com/wtfutil/wtf/modules/spotify"
	"github.com/wtfutil/wtf/modules/spotifyweb"
	"github.com/wtfutil/wtf/modules/status"
	"github.com/wtfutil/wtf/modules/system"
	"github.com/wtfutil/wtf/modules/textfile"
	"github.com/wtfutil/wtf/modules/todo"
	"github.com/wtfutil/wtf/modules/todoist"
	"github.com/wtfutil/wtf/modules/travisci"
	"github.com/wtfutil/wtf/modules/trello"
	"github.com/wtfutil/wtf/modules/twitter"
	"github.com/wtfutil/wtf/modules/unknown"
	"github.com/wtfutil/wtf/modules/victorops"
	"github.com/wtfutil/wtf/modules/weatherservices/prettyweather"
	"github.com/wtfutil/wtf/modules/weatherservices/weather"
	"github.com/wtfutil/wtf/modules/zendesk"
	"github.com/wtfutil/wtf/wtf"
)

var focusTracker wtf.FocusTracker
var runningWidgets []wtf.Wtfable

// Config parses the config.yml file and makes available the settings within
var Config *config.Config

var (
	commit  = "dev"
	date    = "dev"
	version = "dev"
)

/* -------------------- Functions -------------------- */

func disableAllWidgets(widgets []wtf.Wtfable) {
	for _, widget := range widgets {
		widget.Disable()
	}
}

func initializeFocusTracker(app *tview.Application, widgets []wtf.Wtfable) {
	focusTracker = wtf.FocusTracker{
		App:     app,
		Idx:     -1,
		Widgets: widgets,
	}

	focusTracker.AssignHotKeys()
}

func keyboardIntercept(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlR:
		refreshAllWidgets(runningWidgets)
	case tcell.KeyTab:
		focusTracker.Next()
	case tcell.KeyBacktab:
		focusTracker.Prev()
	case tcell.KeyEsc:
		focusTracker.None()
	}

	if focusTracker.FocusOn(string(event.Rune())) {
		return nil
	}

	return event
}

func loadConfigFile(filePath string) {
	Config = cfg.LoadConfigFile(filePath)
	wtf.Config = Config
}

func refreshAllWidgets(widgets []wtf.Wtfable) {
	for _, widget := range widgets {
		go widget.Refresh()
	}
}

func setTerm() {
	err := os.Setenv("TERM", Config.UString("wtf.term", os.Getenv("TERM")))
	if err != nil {
		return
	}
}

func watchForConfigChanges(app *tview.Application, configFilePath string, grid *tview.Grid, pages *tview.Pages) {
	watch := watcher.New()
	absPath, _ := wtf.ExpandHomeDir(configFilePath)

	// Notify write events
	watch.FilterOps(watcher.Write)

	go func() {
		for {
			select {
			case <-watch.Event:
				// Disable all widgets to stop scheduler goroutines and rmeove widgets from memory.
				disableAllWidgets(runningWidgets)

				loadConfigFile(absPath)

				widgets := makeWidgets(app, pages)
				validateWidgets(widgets)

				initializeFocusTracker(app, widgets)

				display := wtf.NewDisplay(widgets)
				pages.AddPage("grid", display.Grid, true, true)
			case err := <-watch.Error:
				log.Fatalln(err)
			case <-watch.Closed:
				return
			}
		}
	}()

	// Watch config file for changes.
	if err := watch.Add(absPath); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := watch.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func makeWidget(app *tview.Application, pages *tview.Pages, widgetName string) wtf.Wtfable {
	var widget wtf.Wtfable

	// Always in alphabetical order
	switch widgetName {
	case "bamboohr":
		cfg := bamboohr.NewSettingsFromYAML(wtf.Config)
		widget = bamboohr.NewWidget(app, cfg)
	case "bargraph":
		widget = bargraph.NewWidget(app)
	case "bittrex":
		cfg := bittrex.NewSettingsFromYAML(wtf.Config)
		widget = bittrex.NewWidget(app, cfg)
	case "blockfolio":
		cfg := blockfolio.NewSettingsFromYAML(wtf.Config)
		widget = blockfolio.NewWidget(app, cfg)
	case "circleci":
		cfg := circleci.NewSettingsFromYAML(wtf.Config)
		widget = circleci.NewWidget(app, cfg)
	case "clocks":
		cfg := clocks.NewSettingsFromYAML(wtf.Config)
		widget = clocks.NewWidget(app, cfg)
	case "cmdrunner":
		cfg := cmdrunner.NewSettingsFromYAML(wtf.Config)
		widget = cmdrunner.NewWidget(app, cfg)
	case "cryptolive":
		cfg := cryptolive.NewSettingsFromYAML(wtf.Config)
		widget = cryptolive.NewWidget(app, cfg)
	case "datadog":
		cfg := datadog.NewSettingsFromYAML(wtf.Config)
		widget = datadog.NewWidget(app, cfg)
	case "gcal":
		cfg := gcal.NewSettingsFromYAML(wtf.Config)
		widget = gcal.NewWidget(app, cfg)
	case "gerrit":
		cfg := gerrit.NewSettingsFromYAML(wtf.Config)
		widget = gerrit.NewWidget(app, pages, cfg)
	case "git":
		widget = git.NewWidget(app, pages)
	case "github":
		widget = github.NewWidget(app, pages)
	case "gitlab":
		widget = gitlab.NewWidget(app, pages)
	case "gitter":
		widget = gitter.NewWidget(app, pages)
	case "gspreadsheets":
		widget = gspreadsheets.NewWidget(app)
	case "hackernews":
		widget = hackernews.NewWidget(app, pages)
	case "ipapi":
		widget = ipapi.NewWidget(app)
	case "ipinfo":
		widget = ipinfo.NewWidget(app)
	case "jenkins":
		widget = jenkins.NewWidget(app, pages)
	case "jira":
		widget = jira.NewWidget(app, pages)
	case "logger":
		widget = logger.NewWidget(app)
	case "mercurial":
		widget = mercurial.NewWidget(app, pages)
	case "nbascore":
		widget = nbascore.NewWidget(app, pages)
	case "newrelic":
		widget = newrelic.NewWidget(app)
	case "opsgenie":
		widget = opsgenie.NewWidget(app)
	case "pagerduty":
		widget = pagerduty.NewWidget(app)
	case "power":
		widget = power.NewWidget(app)
	case "prettyweather":
		widget = prettyweather.NewWidget(app)
	case "resourceusage":
		widget = resourceusage.NewWidget(app)
	case "security":
		widget = security.NewWidget(app)
	case "status":
		widget = status.NewWidget(app)
	case "system":
		widget = system.NewWidget(app, date, version)
	case "spotify":
		widget = spotify.NewWidget(app, pages)
	case "spotifyweb":
		widget = spotifyweb.NewWidget(app, pages)
	case "textfile":
		widget = textfile.NewWidget(app, pages)
	case "todo":
		cfg := todo.NewSettingsFromYAML(wtf.Config)
		widget = todo.NewWidget(app, pages, cfg)
	case "todoist":
		widget = todoist.NewWidget(app, pages)
	case "travisci":
		widget = travisci.NewWidget(app, pages)
	case "rollbar":
		widget = rollbar.NewWidget(app, pages)
	case "trello":
		widget = trello.NewWidget(app)
	case "twitter":
		widget = twitter.NewWidget(app, pages)
	case "victorops":
		widget = victorops.NewWidget(app)
	case "weather":
		widget = weather.NewWidget(app, pages)
	case "zendesk":
		widget = zendesk.NewWidget(app)
	default:
		widget = unknown.NewWidget(app, widgetName)
	}

	return widget
}

func makeWidgets(app *tview.Application, pages *tview.Pages) []wtf.Wtfable {
	widgets := []wtf.Wtfable{}

	mods, _ := Config.Map("wtf.mods")

	for mod := range mods {
		if enabled := Config.UBool("wtf.mods."+mod+".enabled", false); enabled {
			widget := makeWidget(app, pages, mod)
			widgets = append(widgets, widget)
		}
	}

	// This is a hack to allow refreshAllWidgets and disableAllWidgets to work
	// Need to implement a non-global way to track these
	runningWidgets = widgets

	return widgets
}

// Check that all the loaded widgets are valid for display
func validateWidgets(widgets []wtf.Wtfable) {
	for _, widget := range widgets {
		if widget.Enabled() && !widget.IsPositionable() {
			errStr := fmt.Sprintf("Widget config has invalid values: %s", widget.Key())
			log.Fatalln(errStr)
		}
	}
}

/* -------------------- Main -------------------- */

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flags := flags.NewFlags()
	flags.Parse()
	flags.Display(version)

	cfg.MigrateOldConfig()
	cfg.CreateConfigDir()
	cfg.CreateConfigFile()
	loadConfigFile(flags.ConfigFilePath())

	if flags.Profile {
		defer profile.Start(profile.MemProfile).Stop()
	}

	setTerm()

	app := tview.NewApplication()
	pages := tview.NewPages()

	widgets := makeWidgets(app, pages)
	validateWidgets(widgets)

	initializeFocusTracker(app, widgets)

	display := wtf.NewDisplay(widgets)
	pages.AddPage("grid", display.Grid, true, true)

	app.SetInputCapture(keyboardIntercept)

	go watchForConfigChanges(app, flags.Config, display.Grid, pages)

	if err := app.SetRoot(pages, true).Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
