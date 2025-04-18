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
	"yeetfile/cli/styles"
	"yeetfile/cli/utils"
)

const CMD string = "%-12s | %s"

var EXMPL string = "\n" + strings.Repeat(" ", 17) + "- %s"

// Not really necessary, but for readability
var (
	Auth     string = globals.I18n.T("cli.command.auth")
	Signup   string = globals.I18n.T("cli.command.signup")
	Login    string = globals.I18n.T("cli.command.login")
	Logout   string = globals.I18n.T("cli.command.logout")
	Vault    string = globals.I18n.T("cli.command.vault")
	Pass     string = globals.I18n.T("cli.command.pass")
	Send     string = globals.I18n.T("cli.command.send")
	Download string = globals.I18n.T("cli.command.download")
	Account  string = globals.I18n.T("cli.command.account")
	Help     string = globals.I18n.T("cli.command.help")
)

var CommandMap = map[string][]func(){
	globals.I18n.T("cli.command.auth"):     {auth.ShowAuthModel},
	globals.I18n.T("cli.command.signup"):   {signup.ShowSignupModel, login.ShowLoginModel},
	globals.I18n.T("cli.command.login"):    {login.ShowLoginModel},
	globals.I18n.T("cli.command.logout"):   {logout.ShowLogoutModel},
	globals.I18n.T("cli.command.vault"):    {vault.ShowFileVaultModel},
	globals.I18n.T("cli.command.pass"):     {vault.ShowPassVaultModel},
	globals.I18n.T("cli.command.send"):     {send.ShowSendModel},
	globals.I18n.T("cli.command.download"): {download.ShowDownloadModel},
	globals.I18n.T("cli.command.account"):  {account.ShowAccountModel},
	globals.I18n.T("cli.command.help"):     {printHelp},
}

var AuthHelp = []string{
	fmt.Sprintf(CMD, globals.I18n.T("cli.command.signup"), globals.I18n.T("cli.command.signup_help")),
	fmt.Sprintf(CMD, globals.I18n.T("cli.command.login"), globals.I18n.T("cli.command.login_help")),
	fmt.Sprintf(CMD, globals.I18n.T("cli.command.logout"), globals.I18n.T("cli.command.logout_help")),
}

var ActionHelp = []string{
	fmt.Sprintf(CMD,
		globals.I18n.T("cli.command.account"), globals.I18n.T("cli.command.account_help")),
	fmt.Sprintf(CMD+EXMPL,
		globals.I18n.T("cli.command.vault"), globals.I18n.T("cli.command.vault_help"),
		globals.I18n.T("cli.command.vault_exp1")),
	fmt.Sprintf(CMD+EXMPL,
		globals.I18n.T("cli.command.pass"), globals.I18n.T("cli.command.pass_help"),
		globals.I18n.T("cli.command.pass_exp1")),
	fmt.Sprintf(CMD+EXMPL+EXMPL+EXMPL,
		globals.I18n.T("cli.command.send"), globals.I18n.T("cli.command.send_help"),
		globals.I18n.T("cli.command.send_exp1"),
		globals.I18n.T("cli.command.send_exp2"),
		globals.I18n.T("cli.command.send_exp3")),
	fmt.Sprintf(CMD+EXMPL+EXMPL+EXMPL,
		globals.I18n.T("cli.command.download"), globals.I18n.T("cli.command.download_help"),
		globals.I18n.T("cli.command.download_exp1"),
		globals.I18n.T("cli.command.download_exp2"),
		globals.I18n.T("cli.command.download_exp3")),
}

var HelpMsg = "\n" + globals.I18n.T("cli.helpmsg1") + "\n"

var CommandHelpStr = `
  %s`

func printHelp() {
	HelpMsg += "\n" + globals.I18n.T("cli.helpmsg2")
	for _, msg := range AuthHelp {
		HelpMsg += fmt.Sprintf(CommandHelpStr, msg)
	}

	HelpMsg += "\n\n" + globals.I18n.T("cli.helpmsg3")
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
				utils.HandleCLIError(globals.I18n.T("cli.error.noconnect"), err)
				return
			} else if err != nil {
				utils.HandleCLIError(globals.I18n.T("cli.error.initcli"), err)
				return
			}

			styles.PrintErrStr(globals.I18n.T("cli.error.missingcmd"))
			printHelp()
			return
		}
	} else {
		if args[1] == globals.I18n.T("cli.args_1") ||
			args[1] == globals.I18n.T("cli.args_2") || args[1] == globals.I18n.T("cli.args_3") {
			printHelp()
			return
		}
		command = args[1]
	}

	viewFunctions, ok := CommandMap[command]
	if !ok {
		styles.PrintErrStr(fmt.Sprintf(globals.I18n.T("cli.error.invalidcmd"), command))
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
			utils.HandleCLIError(globals.I18n.T("cli.error.noconnect"), authErr)
			return
		} else if !isAuthCommand(command) && command != Download && authErr != nil {
			styles.PrintErrStr(globals.I18n.T("cli.error.notlogin"))
			return
		}
	}

	if !isAuthCommand(command) {
		sessionErr := validateCurrentSession()
		if sessionErr != nil {
			errStr := fmt.Sprintf(globals.I18n.T("cli.error.invsession"), sessionErr)
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
		return errors.New(globals.I18n.T("cli.error.notlogin2"))
	}

	return nil
}

func validateCurrentSession() error {
	cliKey := crypto.ReadCLIKey()
	if cliKey == nil || len(cliKey) == 0 {
		errMsg := fmt.Sprintf(globals.I18n.T("cli.error.missingvar"), crypto.CLIKeyEnvVar)
		return errors.New(errMsg)
	}

	return nil
}

// isAuthCommand checks if the provided command is related to authentication
func isAuthCommand(cmd string) bool {
	return cmd == Login || cmd == Signup || cmd == Logout || cmd == Auth
}
