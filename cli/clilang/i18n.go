package clilang

import (
	"log"
	"yeetfile/cli/config/configbase"
	"yeetfile/lang"
)

var I18n *lang.I18n

func init() {

	var config, err = configbase.LoadConfig()

	var language string
	if len(config.Locale) > 0 {
		language = config.Locale
	} else {
		language = lang.DetectSystemLanguage()
	}
	i18n, err := lang.LoadI18n(language)
	if err != nil {
		log.Println("Error loading language:", err)
		return
	}
	I18n = i18n
}
