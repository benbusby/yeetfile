package commands

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"yeetfile/cli/commands/account"
	"yeetfile/cli/commands/auth"
	"yeetfile/cli/commands/auth/login"
	"yeetfile/cli/commands/auth/logout"
	"yeetfile/cli/commands/auth/signup"
	"yeetfile/cli/commands/download"
	"yeetfile/cli/commands/send"
	"yeetfile/cli/commands/vault"
	"yeetfile/cli/crypto"
	"yeetfile/cli/globals"
	"yeetfile/cli/lang"
	"yeetfile/cli/styles"
	"yeetfile/cli/utils"
)

const CMD string = "%-12s | %s"

var EXMPL string = "\n" + strings.Repeat(" ", 17) + "- %s"

// Not really necessary, but for readability
var (
	Auth     string = lang.I18n.T("cli.command.auth")
	Signup   string = lang.I18n.T("cli.command.signup")
	Login    string = lang.I18n.T("cli.command.login")
	Logout   string = lang.I18n.T("cli.command.logout")
	Vault    string = lang.I18n.T("cli.command.vault")
	Pass     string = lang.I18n.T("cli.command.pass")
	Send     string = lang.I18n.T("cli.command.send")
	Download string = lang.I18n.T("cli.command.download")
	Account  string = lang.I18n.T("cli.command.account")
	Help     string = lang.I18n.T("cli.command.help")
)

var CommandMap = map[string][]func(){
	lang.I18n.T("cli.command.auth"):     {auth.ShowAuthModel},
	lang.I18n.T("cli.command.signup"):   {signup.ShowSignupModel, login.ShowLoginModel},
	lang.I18n.T("cli.command.login"):    {login.ShowLoginModel},
	lang.I18n.T("cli.command.logout"):   {logout.ShowLogoutModel},
	lang.I18n.T("cli.command.vault"):    {vault.ShowFileVaultModel},
	lang.I18n.T("cli.command.pass"):     {vault.ShowPassVaultModel},
	lang.I18n.T("cli.command.send"):     {send.ShowSendModel},
	lang.I18n.T("cli.command.download"): {download.ShowDownloadModel},
	lang.I18n.T("cli.command.account"):  {account.ShowAccountModel},
	lang.I18n.T("cli.command.help"):     {printHelp},
}

var AuthHelp = []string{
	fmt.Sprintf(CMD, lang.I18n.T("cli.command.signup"), lang.I18n.T("cli.command.signup_help")),
	fmt.Sprintf(CMD, lang.I18n.T("cli.command.login"), lang.I18n.T("cli.command.login_help")),
	fmt.Sprintf(CMD, lang.I18n.T("cli.command.logout"), lang.I18n.T("cli.command.logout_help")),
}

var ActionHelp = []string{
	fmt.Sprintf(CMD,
		lang.I18n.T("cli.command.account"), lang.I18n.T("cli.command.account_help")),
	fmt.Sprintf(CMD+EXMPL,
		lang.I18n.T("cli.command.vault"), lang.I18n.T("cli.command.vault_help"),
		lang.I18n.T("cli.command.vault_example")),
	fmt.Sprintf(CMD+EXMPL,
		lang.I18n.T("cli.command.pass"), lang.I18n.T("cli.command.pass_help"),
		lang.I18n.T("cli.command.pass_example")),
	fmt.Sprintf(CMD+EXMPL+EXMPL+EXMPL,
		lang.I18n.T("cli.command.send"), lang.I18n.T("cli.command.send_help"),
		lang.I18n.T("cli.command.send_example_1"),
		lang.I18n.T("cli.command.send_example_2"),
		lang.I18n.T("cli.command.send_example_3")),
	fmt.Sprintf(CMD+EXMPL+EXMPL+EXMPL,
		lang.I18n.T("cli.command.download"), lang.I18n.T("cli.command.download_help"),
		lang.I18n.T("cli.command.download_example_1"),
		lang.I18n.T("cli.command.download_example_2"),
		lang.I18n.T("cli.command.download_example_3")),
}

var HelpMsg = "\n" + lang.I18n.T("cli.help.usage") + "\n"

var CommandHelpStr = `
  %s`

func printHelp() {
	HelpMsg += "\n" + lang.I18n.T("cli.help.auth_cmds")
	for _, msg := range AuthHelp {
		HelpMsg += fmt.Sprintf(CommandHelpStr, msg)
	}

	HelpMsg += "\n\n" + lang.I18n.T("cli.help.action_cmds")
	for _, msg := range ActionHelp {
		HelpMsg += fmt.Sprintf(CommandHelpStr, msg)
	}

	fmt.Println(HelpMsg)
	fmt.Println()
}

// Entrypoint is the main entrypoint to the CLI
func Entrypoint(args []string) {
	var isLoggedIn bool
	var err error
	var command string
	if len(args) < 2 {
		if isLoggedIn, err = auth.IsUserAuthenticated(); !isLoggedIn || err != nil {
			command = Auth
		} else if len(globals.Config.DefaultView) > 0 {
			command = globals.Config.DefaultView
		} else {
			if _, ok := err.(*net.OpError); ok {
				utils.HandleCLIError(lang.I18n.T("cli.error.no_connection"), err)
				return
			} else if err != nil {
				utils.HandleCLIError(lang.I18n.T("cli.error.init_cli"), err)
				return
			}

			styles.PrintErrStr(lang.I18n.T("cli.error.missing_cmd"))
			printHelp()
			return
		}
	} else {
		if args[1] == lang.I18n.T("cli.args.help_short") || args[1] == lang.I18n.T("cli.args.help_long") {
			printHelp()
			return
		}
		command = args[1]
	}

	viewFunctions, ok := CommandMap[command]
	if !ok {
		styles.PrintErrStr(fmt.Sprintf(lang.I18n.T("cli.error.invalid_cmd"), command))
		printHelp()
		return
	} else if command == Help {
		printHelp()
		return
	}

	// Check session state and ensure server is reachable
	if !isLoggedIn && err == nil {
		authErr := validateAuth()
		if _, ok := authErr.(*net.OpError); ok {
			utils.HandleCLIError(lang.I18n.T("cli.error.no_connection"), authErr)
			return
		} else if !isAuthCommand(command) && command != Download && authErr != nil {
			styles.PrintErrStr(lang.I18n.T("cli.error.no_login"))
			return
		}
	}

	if !isAuthCommand(command) {
		sessionErr := validateCurrentSession()
		if sessionErr != nil {
			errStr := fmt.Sprintf(lang.I18n.T("cli.error.bad_session"), sessionErr)
			styles.PrintErrStr(errStr)
			return
		}
	}

	// Set up logging output (can't log to stdout while bubbletea is running)
	var debugFile string
	if len(globals.Config.DebugFile) > 0 && globals.Config.DebugMode {
		homeDir, _ := os.UserHomeDir()
		debugFile = strings.Replace(globals.Config.DebugFile, "~", homeDir, 1)
	} else {
		debugFile = os.DevNull
	}

	f, err := tea.LogToFile(debugFile, "debug")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	// Run view function(s)
	for _, viewFunction := range viewFunctions {
		viewFunction()
	}
}

func validateAuth() error {
	if loggedIn, err := auth.IsUserAuthenticated(); !loggedIn || err != nil {
		if err != nil {
			return err
		}
		return errors.New(lang.I18n.T("cli.error.no_login2"))
	}

	return nil
}

func validateCurrentSession() error {
	cliKey := crypto.ReadCLIKey()
	if cliKey == nil || len(cliKey) == 0 {
		errMsg := fmt.Sprintf(lang.I18n.T("cli.error.missing_var"), crypto.CLIKeyEnvVar)
		return errors.New(errMsg)
	}

	return nil
}

// isAuthCommand checks if the provided command is related to authentication
func isAuthCommand(cmd string) bool {
	return cmd == Login || cmd == Signup || cmd == Logout || cmd == Auth
}
