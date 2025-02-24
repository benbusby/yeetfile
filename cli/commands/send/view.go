package send

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"yeetfile/shared/endpoints"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"

	"yeetfile/cli/globals"
	"yeetfile/cli/styles"
	"yeetfile/cli/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

var (
	fileOption = "File"
	textOption = "Text"
)

var (
	downloads       string
	expiration      string
	expirationUnits string
	password        string
	setPassword     bool
)

var serverError error

var (
	emptySendError      = errors.New("missing file or text to send")
	expValidationError  = errors.New("input must only contain numeric characters")
	inputTooLowError    = errors.New("input must be greater >= 1")
	exceedsMaxDownloads = errors.New("max downloads must be <= 10")
	exceedsMaxTextLen   = errors.New(fmt.Sprintf(
		"text exceeds max length (%d)",
		constants.MaxPlaintextLen))
)

var expExceedsMaxErr = errors.New(fmt.Sprintf(
	"expiration must be < %d days in the future",
	constants.MaxSendAgeDays))

func getSendFields() []huh.Field {
	if globals.Config.Send.Downloads > 0 {
		downloads = strconv.Itoa(globals.Config.Send.Downloads)
	}

	if globals.Config.Send.ExpirationAmount > 0 {
		expiration = strconv.Itoa(globals.Config.Send.ExpirationAmount)
	}

	if len(globals.Config.Send.ExpirationUnits) > 0 {
		expirationUnits = globals.Config.Send.ExpirationUnits
	}

	return []huh.Field{
		huh.NewInput().Title("Expiration").
			Validate(func(s string) error {
				if len(s) == 0 {
					return nil
				}

				val, err := strconv.Atoi(s)
				if err != nil {
					return expValidationError
				} else if val < 1 {
					return inputTooLowError
				} else if !isValidExp(int64(val), expirationUnits) {
					return expExceedsMaxErr
				}

				return nil
			}).Value(&expiration),
		huh.NewSelect[string]().Title("Units").
			Options([]huh.Option[string]{
				huh.NewOption("Minutes", expMinutes),
				huh.NewOption("Hours", expHours),
				huh.NewOption("Days", expDays),
			}...).Value(&expirationUnits),
		huh.NewInput().Title("Max Downloads").
			Validate(func(s string) error {
				if len(s) == 0 {
					return nil
				}

				val, err := strconv.Atoi(s)
				if err != nil {
					return expValidationError
				} else if val > 10 {
					return exceedsMaxDownloads
				} else if val < 1 {
					return inputTooLowError
				}

				return nil
			}).Value(&downloads),
		huh.NewSelect[bool]().Title("Set Password (Optional)").
			Description("If set to 'Yes', you will be prompted to enter\n" +
				"a password on the next screen.").
			Options([]huh.Option[bool]{
				huh.NewOption("No", false),
				huh.NewOption("Yes", true),
			}...).Value(&setPassword),
	}
}

func getPasswordGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Password").
			EchoMode(huh.EchoModePassword).
			Value(&password),
		huh.NewInput().Title("Confirm Password").
			EchoMode(huh.EchoModePassword).Validate(
			func(s string) error {
				if s != password {
					return errors.New("passwords don't match")
				}

				return nil
			}),
	).WithHideFunc(func() bool {
		return !setPassword
	})
}

func getConfirmationField(toValidate *string) huh.Field {
	return huh.NewConfirm().Title("Create Link").DescriptionFunc(
		func() string {
			exp := expiration
			d := downloads
			units := expirationUnits
			dStr := "downloads"

			if len(exp) == 0 {
				exp = "--"
			} else if exp == "1" {
				units = units[:len(units)-1]
			}

			expInt, _ := strconv.Atoi(exp)

			if len(d) == 0 {
				d = "--"
			} else if d == "1" {
				dStr = dStr[:len(dStr)-1]
			}

			msg := fmt.Sprintf(
				"The link will expire in %s %s (~ %s), "+
					"or after %s %s.",
				exp,
				units,
				getExpString(int64(expInt), expirationUnits),
				d,
				dStr)

			if serverError != nil {
				msg += styles.ErrStyle.Render(
					fmt.Sprintf("\n\nError: %s",
						serverError.Error()))
			}

			return msg
		}, []*string{&expiration, &expirationUnits, &downloads}).
		Affirmative("Create").
		Negative("").Validate(
		func(b bool) error {
			expVal, _ := strconv.Atoi(expiration)
			if len(*toValidate) == 0 {
				return emptySendError
			} else if !isValidExp(int64(expVal), expirationUnits) {
				return expExceedsMaxErr
			}

			return nil
		})
}

func showSendFileModel(filepath string) {
	title := huh.NewNote().Title(utils.GenerateTitle("Send File"))
	filepicker := huh.NewFilePicker().Title("File").Value(&filepath)
	confirm := getConfirmationField(&filepath)
	fields := getSendFields()
	fields = append([]huh.Field{title, filepicker}, fields...)
	fields = append(fields, confirm)

	err := huh.NewForm(huh.NewGroup(fields...), getPasswordGroup()).
		WithTheme(styles.Theme).
		WithShowHelp(true).Run()
	if err != nil {
		return
	}

	var result string
	var secret string
	progress := spinner.New()
	_ = progress.Title("Preparing file...").Action(func() {
		expVal, _ := strconv.Atoi(expiration)
		maxDownloads, _ := strconv.Atoi(downloads)
		result, secret, err = createFileLink(fileUpload{
			FilePath:     filepath,
			ExpUnits:     expirationUnits,
			ExpValue:     expVal,
			Password:     password,
			MaxDownloads: maxDownloads,
		}, func(chunk int, total int) {
			percentage := int((float32(chunk) / float32(total)) * 100)
			msg := fmt.Sprintf("Uploading... (%d%%)", percentage)
			progress.Title(msg)
		})
	}).Run()

	if err != nil {
		serverError = err
		showSendFileModel(filepath)
		return
	}

	showLinkModel("File Link", result, secret)
}

func showSendTextModel(text string) {
	title := huh.NewNote().Title(utils.GenerateTitle("Send Text"))
	input := huh.NewText().Title("Text").
		CharLimit(constants.MaxPlaintextLen).
		Description(fmt.Sprintf("(%d / %d)",
			len(text), constants.MaxPlaintextLen)).
		DescriptionFunc(
			func() string {
				msg := fmt.Sprintf("(%d / %d)",
					len(text), constants.MaxPlaintextLen)
				return msg
			}, &text).
		Validate(func(s string) error {
			if len(s) > constants.MaxPlaintextLen {
				return exceedsMaxTextLen
			}

			return nil
		}).Value(&text)
	confirm := getConfirmationField(&text)
	fields := getSendFields()
	fields = append([]huh.Field{title, input}, fields...)
	fields = append(fields, confirm)

	err := huh.NewForm(huh.NewGroup(fields...), getPasswordGroup()).
		WithTheme(styles.Theme).
		WithShowHelp(true).Run()
	if err != nil {
		return
	}

	var result string
	var secret string
	progress := spinner.New()
	_ = progress.Title("Preparing text...").Action(func() {
		expVal, _ := strconv.Atoi(expiration)
		maxDownloads, _ := strconv.Atoi(downloads)
		result, secret, err = createTextLink(textUpload{
			Text:         text,
			ExpUnits:     expirationUnits,
			ExpValue:     expVal,
			Password:     password,
			MaxDownloads: maxDownloads,
		})
	}).Run()

	if err != nil {
		serverError = err
		showSendTextModel(text)
		return
	}

	showLinkModel("Text Link", result, secret)
}

func showLinkModel(title, id, secret string) {
	resource := fmt.Sprintf("%s#%s",
		shared.EscapeString(id),
		shared.EscapeString(secret))
	link := endpoints.HTMLSendDownload.Format(globals.Config.Server, resource)

	err := huh.NewForm(huh.NewGroup(
		huh.NewNote().Title(utils.GenerateTitle(title)),
		huh.NewNote().
			Title("Link (Web)").
			Description(link),
		huh.NewNote().
			Title("Resource ID (CLI)").
			Description(resource),
		huh.NewNote().
			Title("Note").
			Description("This link will not be shown again. Please"+
				" copy it down now."),
		huh.NewConfirm().Affirmative("OK").Negative(""),
	)).WithTheme(styles.Theme).Run()

	log.Println(err)
}

func ShowSendModel() {
	var filepath string
	var text string
	if len(os.Args) > 2 {
		if _, err := os.Stat(os.Args[2]); err != nil {
			text = strings.Join(os.Args[2:], " ")
		} else {
			filepath = os.Args[2]
		}
	}

	if len(filepath) > 0 {
		showSendFileModel(filepath)
		return
	} else if len(text) > 0 {
		showSendTextModel(text)
		return
	}

	var option int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title(utils.GenerateTitle("Send")),
			huh.NewSelect[int]().Title("Type").Options(
				[]huh.Option[int]{
					huh.NewOption(fileOption, 0),
					huh.NewOption(textOption, 1),
				}...).
				Value(&option),
		),
	).WithTheme(styles.Theme).WithShowHelp(true).Run()
	if err != nil {
		return
	}

	if option == 0 {
		showSendFileModel("")
	} else {
		showSendTextModel("")
	}
}
