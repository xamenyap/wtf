package bittrex

import (
	"encoding/json"
	"fmt"
	"time"

	"net/http"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/wtf"
)

var ok = true
var errorText = ""

const baseURL = "https://bittrex.com/api/v1.1/public/getmarketsummary"

// Widget define wtf widget to register widget later
type Widget struct {
	wtf.TextWidget

	settings *Settings
	summaryList
}

// NewWidget Make new instance of widget
func NewWidget(app *tview.Application, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: wtf.NewTextWidget(app, "Bittrex", "bittrex", false),

		settings:    settings,
		summaryList: summaryList{},
	}

	ok = true
	errorText = ""

	widget.setSummaryList()

	return &widget
}

func (widget *Widget) setSummaryList() {
	sCurrencies := widget.settings.summary
	for baseCurrencyName := range sCurrencies {
		displayName, _ := wtf.Config.String("wtf.mods.bittrex.summary." + baseCurrencyName + ".displayName")
		mCurrencyList := makeSummaryMarketList(baseCurrencyName)
		widget.summaryList.addSummaryItem(baseCurrencyName, displayName, mCurrencyList)
	}
}

func makeSummaryMarketList(currencyName string) []*mCurrency {
	mCurrencyList := []*mCurrency{}

	configMarketList, _ := wtf.Config.List("wtf.mods.bittrex.summary." + currencyName + ".market")
	for _, mCurrencyName := range configMarketList {
		mCurrencyList = append(mCurrencyList, makeMarketCurrency(mCurrencyName.(string)))
	}

	return mCurrencyList
}

func makeMarketCurrency(name string) *mCurrency {
	return &mCurrency{
		name: name,
		summaryInfo: summaryInfo{
			High:           "",
			Low:            "",
			Volume:         "",
			Last:           "",
			OpenBuyOrders:  "",
			OpenSellOrders: "",
		},
	}
}

/* -------------------- Exported Functions -------------------- */

// Refresh & update after interval time
func (widget *Widget) Refresh() {
	widget.updateSummary()
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) updateSummary() {
	// In case if anything bad happened!
	defer func() {
		recover()
	}()

	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	for _, baseCurrency := range widget.summaryList.items {
		for _, mCurrency := range baseCurrency.markets {
			request := makeRequest(baseCurrency.name, mCurrency.name)
			response, err := client.Do(request)

			if err != nil {
				ok = false
				errorText = "Please Check Your Internet Connection!"
				break
			} else {
				ok = true
				errorText = ""
			}

			if response.StatusCode != http.StatusOK {
				errorText = response.Status
				ok = false
				break
			} else {
				ok = true
				errorText = ""
			}

			defer response.Body.Close()
			jsonResponse := summaryResponse{}
			decoder := json.NewDecoder(response.Body)
			decoder.Decode(&jsonResponse)

			if !jsonResponse.Success {
				ok = false
				errorText = fmt.Sprintf("%s-%s: %s", baseCurrency.name, mCurrency.name, jsonResponse.Message)
				break
			}
			ok = true
			errorText = ""

			mCurrency.Last = fmt.Sprintf("%f", jsonResponse.Result[0].Last)
			mCurrency.High = fmt.Sprintf("%f", jsonResponse.Result[0].High)
			mCurrency.Low = fmt.Sprintf("%f", jsonResponse.Result[0].Low)
			mCurrency.Volume = fmt.Sprintf("%f", jsonResponse.Result[0].Volume)
			mCurrency.OpenBuyOrders = fmt.Sprintf("%d", jsonResponse.Result[0].OpenBuyOrders)
			mCurrency.OpenSellOrders = fmt.Sprintf("%d", jsonResponse.Result[0].OpenSellOrders)
		}
	}

	widget.display()
}

func makeRequest(baseName, marketName string) *http.Request {
	url := fmt.Sprintf("%s?market=%s-%s", baseURL, baseName, marketName)
	request, _ := http.NewRequest("GET", url, nil)

	return request
}
