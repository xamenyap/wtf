package cryptolive

import (
	"fmt"
	"sync"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/modules/cryptoexchanges/cryptolive/price"
	"github.com/wtfutil/wtf/modules/cryptoexchanges/cryptolive/toplist"
	"github.com/wtfutil/wtf/wtf"
)

// Widget define wtf widget to register widget later
type Widget struct {
	wtf.TextWidget

	priceWidget   *price.Widget
	toplistWidget *toplist.Widget
	settings      *Settings
}

// NewWidget Make new instance of widget
func NewWidget(app *tview.Application, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: wtf.NewTextWidget(app, "CryptoLive", "cryptolive", false),

		priceWidget:   price.NewWidget(settings.priceSettings),
		toplistWidget: toplist.NewWidget(settings.toplistSettings),
		settings:      settings,
	}

	widget.priceWidget.RefreshInterval = widget.RefreshInterval()
	widget.toplistWidget.RefreshInterval = widget.RefreshInterval()

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh & update after interval time
func (widget *Widget) Refresh() {
	var wg sync.WaitGroup

	wg.Add(2)
	widget.priceWidget.Refresh(&wg)
	widget.toplistWidget.Refresh(&wg)
	wg.Wait()

	display(widget)
}

/* -------------------- Unexported Functions -------------------- */

func display(widget *Widget) {
	str := ""
	str += widget.priceWidget.Result
	str += widget.toplistWidget.Result
	widget.View.SetText(fmt.Sprintf("\n%s", str))
}
