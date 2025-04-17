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

const CMD string = "%-12s"

// Not really necessary, but for readability
var (
	Auth     string = "auth"
	Signup   string = globals.I18n.T("cli.command.signup")
	Login    string = globals.I18n.T("cli.command.login")
	Logout   string = globals.I18n.T("cli.command.logout")
	Vault    string = "vault"
	Pass     string = "pass"
	Send     string = "send"
	Download string = "download"
	Account  string = "account"
	Help     string = "help"
)

var CommandMap = map[string][]func(){
	Auth:                                 {auth.ShowAuthModel},
	globals.I18n.T("cli.command.signup"): {signup.ShowSignupModel, login.ShowLoginModel},
	globals.I18n.T("cli.command.login"):  {login.ShowLoginModel},
	globals.I18n.T("cli.command.logout"): {logout.ShowLogoutModel},
	Vault:                                {vault.ShowFileVaultModel},
	Pass:                                 {vault.ShowPassVaultModel},
	Send:                                 {send.ShowSendModel},
	Download:                             {download.ShowDownloadModel},
	Account:                              {account.ShowAccountModel},
	Help:                                 {printHelp},
}

var AuthHelp = []string{
	fmt.Sprintf(CMD+" | %s", globals.I18n.T("cli.command.signup"), globals.I18n.T("cli.command.signup_help")),
	fmt.Sprintf(CMD+" | %s", globals.I18n.T("cli.command.login"), globals.I18n.T("cli.command.login_help")),
	fmt.Sprintf(CMD+" | %s", globals.I18n.T("cli.command.logout"), globals.I18n.T("cli.command.logout_help")),
}

var ActionHelp = []string{
	fmt.Sprintf("%s  | Manage your YeetFile account", Account),
	fmt.Sprintf("%s    | Manage files and folders in your YeetFile Vault\n"+
		"             - Example: yeetfile vault", Vault),
	fmt.Sprintf("%s     | Manage passwords in your YeetFile Password Vault\n"+
		"             - Example: yeetfile pass", Pass),
	fmt.Sprintf("%s     | Create an end-to-end encrypted shareable link to a file or text\n"+
		"             - Example: yeetfile send\n"+
		"             - Example: yeetfile send path/to/file.png\n"+
		"             - Example: yeetfile send 'top secret text'", Send),
	fmt.Sprintf("%s | Download a file or text uploaded via YeetFile Send\n"+
		"             - Example: yeetfile download\n"+
		"             - Example: yeetfile download https://yeetfile.com/file_abc#top.secret.hash8\n"+
		"             - Example: yeetfile download file_abc#top.secret.hash8", Download),
}

var HelpMsg = `
Usage: yeetfile <command> [args]
`

var CommandHelpStr = `
  %s`

func printHelp() {
	HelpMsg += `
Auth Commands:`
	for _, msg := range AuthHelp {
		HelpMsg += fmt.Sprintf(CommandHelpStr, msg)
	}

	HelpMsg += `

Action Commands:`
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
				utils.HandleCLIError("Unable to connect to the server", err)
				return
			} else if err != nil {
				utils.HandleCLIError("Error initializing CLI tool", err)
				return
			}

			styles.PrintErrStr("-- Missing command")
			printHelp()
			return
		}
	} else {
		if args[1] == "-h" || args[1] == "--help" || args[1] == "help" {
			printHelp()
			return
		}
		command = args[1]
	}

	viewFunctions, ok := CommandMap[command]
	if !ok {
		styles.PrintErrStr(fmt.Sprintf("-- Invalid command '%s'", command))
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
			utils.HandleCLIError("Unable to connect to the server", authErr)
			return
		} else if !isAuthCommand(command) && command != Download && authErr != nil {
			styles.PrintErrStr("You are not logged in. " +
				"Use the 'login' or 'signup' commands to continue.")
			return
		}
	}

	if !isAuthCommand(command) {
		sessionErr := validateCurrentSession()
		if sessionErr != nil {
			errStr := fmt.Sprintf("Error validating session: %v", sessionErr)
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
		return errors.New("not logged in")
	}

	return nil
}

func validateCurrentSession() error {
	cliKey := crypto.ReadCLIKey()
	if cliKey == nil || len(cliKey) == 0 {
		errMsg := fmt.Sprintf(`Missing '%[1]s' environment variable.
You must include the value returned for this variable either in your shell
config file (.bashrc, .zshrc, etc), run 'export %[1]s=xxxx' in your current 
session, or prefix commands with the variable (i.e. %[1]s=xxxx yeetfile vault)`,
			crypto.CLIKeyEnvVar)
		return errors.New(errMsg)
	}

	return nil
}

// isAuthCommand checks if the provided command is related to authentication
func isAuthCommand(cmd string) bool {
	return cmd == Login || cmd == Signup || cmd == Logout || cmd == Auth
}
