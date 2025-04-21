package lang

import (
	"log"
	"yeetfile/cli/config"
	"yeetfile/shared"
)

var I18n *shared.I18n

func init() {

	var config = config.LoadConfig()

	var lang string
	if len(config.Locale) > 0 {
		lang = config.Locale
	} else {
		lang = shared.DetectSystemLanguage()
	}
	i18n, err := shared.LoadI18n(lang)
	if err != nil {
		log.Println("Error loading language:", err)
		return
	}
	I18n = i18n
}
