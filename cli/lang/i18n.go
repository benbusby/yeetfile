package lang

import (
	"log"
	"yeetfile/cli/config/configfile"
	"yeetfile/shared/lang"
)

var I18n *lang.I18n

func init() {

	var config, err = configfile.LoadConfig()

	var language string
	if len(config.Locale) > 0 {
		language = config.Locale
	} else {
		language = lang.DetectSystemLanguage()
	}
	i18n, err := lang.LoadI18n("cli", language)
	if err != nil {
		log.Println("Error loading language:", err)
		return
	}
	I18n = i18n
}
