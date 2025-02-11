package signup

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/qeesung/image2ascii/convert"
	"image"
	"log"
	"strings"
	"yeetfile/cli/api"
	"yeetfile/cli/globals"
	"yeetfile/cli/styles"
	"yeetfile/cli/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

const signupEmail = "Email Address"
const signupIDOnly = "Account ID Only"
const idOnlyWarning = `
Note: By signing up with an account number only, you will be unable to recover 
your account if you ever lose your account number.`
const showIDMessage = `Your account ID is: %s -- write this down!
This is what you will use to log in, and will not be shown again.`

// ShowSignupModel is the main entrypoint to the YeetFile signup process
func ShowSignupModel() {
	var email string
	var password string
	var passwordHint string
	var signupType string

	var options []huh.Option[string]
	if globals.ServerInfo.EmailConfigured {
		options = huh.NewOptions(signupEmail, signupIDOnly)
	} else {
		options = huh.NewOptions(signupIDOnly)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Sign Up")),
			huh.NewSelect[string]().
				Options(options...).
				Title("Account Type").Value(&signupType),
		),
		huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Sign Up > Email")),
			huh.NewInput().Title("Email").
				Value(&email).Validate(func(s string) error {
				isValid := strings.Contains(s, "@") &&
					strings.Contains(s, ".")
				if isValid {
					return nil
				}

				return errors.New("invalid email")
			}),
			huh.NewInput().Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&password),
			huh.NewInput().Title("Confirm Password").
				EchoMode(huh.EchoModePassword).
				Validate(func(s string) error {
					if s != password {
						return errors.New("passwords do not match")
					}

					return nil
				}),
			huh.NewText().
				Title("Password Hint (optional)").
				Description("Note: Passwords cannot be recovered if lost.\n"+
					"Setting a password hint is recommended.").
				Lines(2).
				Value(&passwordHint),
			huh.NewConfirm().Affirmative("Submit").Negative(""),
		).WithHideFunc(func() bool {
			return signupType != signupEmail
		}),
		huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Sign Up > ID Only")).
				Description(idOnlyWarning),
			huh.NewInput().Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&password),
			huh.NewInput().Title("Confirm Password").
				EchoMode(huh.EchoModePassword).
				Validate(func(s string) error {
					if s == password {
						return nil
					}

					return errors.New("passwords do not match")
				}),
			huh.NewConfirm().Affirmative("Submit").Negative(""),
		).WithHideFunc(func() bool {
			return signupType != signupIDOnly
		}),
	).WithTheme(styles.Theme).WithShowHelp(true).Run()
	utils.HandleCLIError("", err)

	if signupType == signupIDOnly {
		showIDOnlySignupModel(password, "")
	} else if signupType == signupEmail {
		showEmailSignupModel(email, password, passwordHint, "")
	}
}

// showEmailSignupModel shows a spinner while the user's account is created
// and finalized.
func showEmailSignupModel(email, password, hint, serverPw string) {
	var signupErr error
	err := spinner.New().Title("Creating account...").Action(
		func() {
			signup := CreateSignupRequest(email, password, hint, serverPw)
			_, signupErr = globals.API.SubmitSignup(signup)
		}).Run()
	utils.HandleCLIError("", err)

	if signupErr == api.ServerPasswordError {
		serverPassword := showServerPasswordPrompt()
		showEmailSignupModel(email, password, hint, serverPassword)
		return
	}

	utils.HandleCLIError("error creating account", signupErr)

	var code string
	desc := fmt.Sprintf(
		"A verification code has been sent to %s, please enter it below.",
		email)

	var runFunc func(...string)
	runFunc = func(errorMessages ...string) {
		if len(errorMessages) > 0 {
			desc = styles.ErrStyle.Render("Error: " + errorMessages[0])
		}
		err = huh.NewForm(huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Verify Email")).
				Description(desc),
			huh.NewInput().Title("Verification Code").Value(&code).
				Validate(func(s string) error {
					if len(s) == constants.VerificationCodeLength {
						return nil
					}

					msg := fmt.Sprintf(
						"Verification code must be %d-digits",
						constants.VerificationCodeLength)

					return errors.New(msg)
				}),
			huh.NewConfirm().Affirmative("Submit").Negative(""),
		)).WithTheme(styles.Theme).Run()
		utils.HandleCLIError("error verifying email", err)

		var verifyErr error
		err = spinner.New().Title("Verifying account...").Action(
			func() {
				verifyErr = globals.API.VerifyEmail(email, code)
			}).Run()
		utils.HandleCLIError("", err)

		if verifyErr != nil {
			runFunc(verifyErr.Error())
		}
	}

	runFunc()

	err = huh.NewForm(huh.NewGroup(
		huh.NewNote().Title(utils.GenerateTitle("Signup Complete")).
			Description("You may now log in!"),
		huh.NewConfirm().Affirmative("Log In").Negative(""))).
		WithTheme(styles.Theme).Run()
	utils.HandleCLIError("", err)
}

// showIDOnlySignupModel shows a spinner while the user's ID-only account is
// created and finalized.
func showIDOnlySignupModel(password, serverPw string) {
	var response shared.SignupResponse
	var signupErr error
	err := spinner.New().Title("Creating account...").Action(
		func() {
			// Submit blank signup form to indicate an account ID
			// only signup
			response, signupErr = globals.API.SubmitSignup(
				shared.Signup{
					Identifier:              "",
					LoginKeyHash:            nil,
					PublicKey:               nil,
					ProtectedPrivateKey:     nil,
					ProtectedVaultFolderKey: nil,
					ServerPassword:          serverPw,
				},
			)
		}).Run()
	utils.HandleCLIError("", err)

	if signupErr == api.ServerPasswordError {
		serverPassword := showServerPasswordPrompt()
		showIDOnlySignupModel(password, serverPassword)
		return
	}

	utils.HandleCLIError("error creating account", signupErr)

	if len(response.Captcha) > 0 {
		var runFunc func(...string)
		runFunc = func(errorMessages ...string) {
			verificationCode := showCaptchaModel(
				response.Captcha,
				errorMessages...)
			var verifyErr error
			err = spinner.New().Title("Verifying account...").Action(
				func() {
					verify := CreateVerificationRequest(
						response.Identifier,
						password,
						verificationCode)
					verifyErr = globals.API.VerifyAccount(verify)
				}).Run()
			utils.HandleCLIError("", err)
			if verifyErr != nil && verifyErr != huh.ErrUserAborted {
				runFunc(verifyErr.Error())
			}

			showAccountConfirmationModel(response.Identifier)
		}

		runFunc()
	}
}

// showCaptchaModel displays the 6-digit verification code image sent by
// the server as ASCII art in the terminal, and returns the value that the
// user enters.
func showCaptchaModel(captchaStr string, errorMessages ...string) string {
	captchaBytes, err := base64.StdEncoding.DecodeString(captchaStr)
	if err != nil {
		log.Printf("Err: %v\n", err)
		return ""
	}

	img, _, _ := image.Decode(bytes.NewReader(captchaBytes))

	converter := convert.NewImageConverter()
	options := convert.DefaultOptions
	options.Colored = false
	options.FixedHeight = 40
	options.FixedWidth = 80

	codeStr := converter.Image2ASCIIString(img, &options)
	codeStr = shared.TrimEmptyLines(codeStr)

	var code string

	desc := fmt.Sprintf("Must be %d-digits", constants.VerificationCodeLength)
	if len(errorMessages) > 0 {
		desc = styles.ErrStyle.Render(errorMessages[0])
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Verification Code").
				Description(codeStr),
			huh.NewInput().
				Title("Enter Code").
				Description(desc).
				Value(&code),
		),
	).WithTheme(styles.Theme).Run()
	utils.HandleCLIError("", err)

	return strings.TrimSpace(code)
}

// showAccountConfirmationModel displays the user's new account ID to the user
func showAccountConfirmationModel(id string) {
	msg := fmt.Sprintf(showIDMessage, id)
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Your Account ID")).
				Description(msg),
			huh.NewConfirm().
				Title("Warning").
				Description("Do not continue without writing "+
					"your account ID down. This will not "+
					"be shown again.").
				Affirmative("Log In").Negative(""),
		),
	).WithTheme(styles.Theme).WithShowHelp(true).Run()
	utils.HandleCLIError("error showing confirmation", err)
}

func showServerPasswordPrompt() string {
	var serverPw string
	msg := fmt.Sprintf("This server (%s) is password protected.\nPlease enter"+
		" the server password below, or use a different server.",
		globals.Config.Server)
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Server Password")).
				Description(msg),
			huh.NewInput().Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&serverPw),
			huh.NewConfirm().Affirmative("Submit").Negative(""),
		),
	).WithTheme(styles.Theme).WithShowHelp(true).Run()
	utils.HandleCLIError("", err)
	return serverPw
}
