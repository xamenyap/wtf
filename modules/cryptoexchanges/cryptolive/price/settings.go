package price

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

type colors struct {
	from struct {
		name        string
		displayName string
	}
	to struct {
		name  string
		price string
	}
	top struct {
		from struct {
			name        string
			displayName string
		}
		to struct {
			name  string
			field string
			value string
		}
	}
}

type Settings struct {
	colors
	common     *cfg.Common
	currencies map[string]interface{}
	top        map[string]interface{}
}

func NewSettingsFromYAML(ymlConfig *config.Config) *Settings {
	localConfig, _ := ymlConfig.Get("wtf.mods.cryptolive")

	currencies, _ := localConfig.Map("currencies")
	top, _ := localConfig.Map("top")

	settings := Settings{
		common:     cfg.NewCommonSettingsFromYAML(ymlConfig),
		currencies: currencies,
		top:        top,
	}

	settings.colors.from.name = localConfig.UString("colors.from.name")
	settings.colors.from.displayName = localConfig.UString("colors.from.displayName")

	settings.colors.to.name = localConfig.UString("colors.to.name")
	settings.colors.to.price = localConfig.UString("colors.to.price")

	settings.colors.top.from.name = localConfig.UString("colors.top.from.name")
	settings.colors.top.from.displayName = localConfig.UString("colors.top.from.displayName")

	settings.colors.top.to.name = localConfig.UString("colors.top.to.name")
	settings.colors.top.to.field = localConfig.UString("colors.top.to.field")
	settings.colors.top.to.value = localConfig.UString("colors.top.to.value")

	return &settings
}
