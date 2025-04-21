package lang

import (
	"log"
	//"yeetfile/cli/config"
	"yeetfile/shared"
)

var I18n *shared.I18n

func init() {

	/*
		Do not use config anymore to overwrite locale setting to keep
		lang dependency free and prevent import cycles
		instead force langage via Shell: LANG=en_US.UTF-8 -/yeetfile -h
		Can easily be set through wrapper script or alias
	*/
	/*
		var config = config.LoadConfig()

		var lang string
		if len(config.Locale) > 0 {
			lang = config.Locale
		} else {
			lang = shared.DetectSystemLanguage()
		}
	*/
	var lang = shared.DetectSystemLanguage()
	i18n, err := shared.LoadI18n(lang)
	if err != nil {
		log.Println("Error loading language:", err)
		return
	}
	I18n = i18n
}
