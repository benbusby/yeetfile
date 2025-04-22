package utils

import (
	"fmt"
	"os"
	"yeetfile/cli/lang"
	"yeetfile/cli/styles"

	"github.com/charmbracelet/huh"
)

func HandleCLIError(msg string, err error) {
	if err == nil {
		return
	} else if err == huh.ErrUserAborted {
		os.Exit(0)
	}

	styles.PrintErrStr(fmt.Sprintf(lang.I18n.T("cli.utils.error.errtitle")+": %s - %v\n", msg, err))
	os.Exit(1)
}
