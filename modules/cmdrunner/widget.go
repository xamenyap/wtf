package cmdrunner

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/wtf"
)

type Widget struct {
	wtf.TextWidget

	args     []string
	cmd      string
	result   string
	settings *Settings
}

func NewWidget(app *tview.Application, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: wtf.NewTextWidget(app, "CmdRunner", "cmdrunner", false),

		args:     settings.args,
		cmd:      settings.cmd,
		settings: settings,
	}

	widget.View.SetWrap(true)

	return &widget
}

func (widget *Widget) Refresh() {
	widget.execute()

	title := tview.TranslateANSI(wtf.Config.UString("wtf.mods.cmdrunner.title", widget.String()))
	widget.View.SetTitle(title)

	widget.View.SetText(widget.result)
}

func (widget *Widget) String() string {
	args := strings.Join(widget.args, " ")

	if args != "" {
		return fmt.Sprintf(" %s %s ", widget.cmd, args)
	}

	return fmt.Sprintf(" %s ", widget.cmd)
}

func (widget *Widget) execute() {
	cmd := exec.Command(widget.cmd, widget.args...)
	widget.result = tview.TranslateANSI(wtf.ExecuteCommand(cmd))
}
