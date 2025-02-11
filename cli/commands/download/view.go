package download

import (
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"os"
	"strings"
	"yeetfile/cli/globals"
	"yeetfile/cli/styles"
	"yeetfile/cli/transfer"
	"yeetfile/cli/utils"
)

var downloadLink string
var downloadErr error
var saveErr error

func ShowDownloadModel() {
	var err error
	var msg string
	if downloadErr != nil {
		msg = styles.ErrStyle.Render(downloadErr.Error())
	}

	if len(os.Args) < 3 || len(downloadLink) > 0 {
		err = huh.NewForm(huh.NewGroup(
			huh.NewNote().
				Title(utils.GenerateTitle("Download")),
			huh.NewInput().
				Title("Resource ID or URL").
				Placeholder("https://yeetfile.com/send/... | "+
					"file_id#top.secret.hash8").
				Value(&downloadLink),
			huh.NewConfirm().
				Description(msg).
				Affirmative("OK").
				Negative(""),
		)).WithTheme(styles.Theme).Run()
	} else {
		downloadLink = os.Args[2]
	}

	if err != nil {
		return
	}

	if !strings.HasPrefix(downloadLink, "http") {
		downloadSegments := strings.Split(downloadLink, "/")
		if len(downloadSegments) == 1 {
			downloadLink = globals.Config.Server + "/" + downloadLink
		}
	}

	startDownload(downloadLink)
}

func startDownload(link string) {
	preparedDownload, err := prepDownload(link)

	if err != nil {
		downloadErr = err
		ShowDownloadModel()
		return
	}

	showPreviewModel(preparedDownload)
}

func showPreviewModel(prep PreparedDownload) {
	filename := prep.Name
	description := generateDescription(prep)

	overwriteWarning := "Warning: This will overwrite a file with the same " +
		"name in this directory!"

	var downloadHelper huh.Field
	if prep.IsText {
		downloadHelper = huh.NewNote().
			Title("Note").
			Description("The text content will appear in the console" +
				" after downloading.")
	} else {
		downloadHelper = huh.NewInput().Title("Save as...").Value(&filename)
	}

	err := huh.NewForm(huh.NewGroup(
		huh.NewNote().
			Title("Pending Download").
			Description(description),
		downloadHelper,
		huh.NewConfirm().
			Title("Download").
			Affirmative("Start Download").
			Negative("").
			DescriptionFunc(func() string {
				desc := ""
				if _, err := os.Stat(filename); err == nil {
					desc += overwriteWarning
				}

				if saveErr != nil {
					desc += "\n" + styles.ErrStyle.Render(
						saveErr.Error())
				}

				return desc
			}, &filename),
	)).WithTheme(styles.Theme).Run()
	if err != nil {
		return
	}

	if prep.IsText {
		showDownloadTextModel(prep)
	} else {
		showDownloadFileModel(prep, filename)
	}
}

func showPasswordPromptModel(err error) (string, error) {
	desc := "This content is password protected"
	if err != nil {
		desc = styles.ErrStyle.Render(err.Error())
	}

	var password string
	pErr := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Password").
			Description(desc).
			EchoMode(huh.EchoModePassword).
			Value(&password),
		huh.NewConfirm().Affirmative("Submit").Negative(""),
	)).WithTheme(styles.Theme).Run()

	return password, pErr
}

func showDownloadTextModel(prep PreparedDownload) {
	var data []byte
	var err error
	_ = spinner.New().Title("Downloading text...").Action(func() {
		data, err = transfer.DownloadText(prep.ID, prep.Server, prep.Key)
		if err != nil {
			saveErr = err
			return
		}
	}).Run()

	_ = huh.NewForm(huh.NewGroup(
		huh.NewNote().
			Title("Downloaded Text").
			Description(string(data)),
		huh.NewConfirm().Affirmative("Exit").Negative(""),
	)).WithTheme(styles.Theme).Run()
}

func showDownloadFileModel(prep PreparedDownload, filename string) {
	downloadSpinner := spinner.New()
	_ = downloadSpinner.Title("Downloading file...").Action(func() {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			saveErr = err
			return
		}

		p := transfer.InitSendDownload(
			prep.ID,
			prep.Server,
			prep.Key,
			file,
			prep.Chunks,
		)

		chunk := 0
		saveErr = p.DownloadData(func() {
			progress := int((float32(chunk) / float32(p.NumChunks)) * 100)
			msg := fmt.Sprintf("Downloading file... (%d%%)", progress)
			downloadSpinner.Title(msg)
			chunk++
		})
	}).Run()

	if saveErr != nil {
		showPreviewModel(prep)
		return
	}

	fmt.Printf("\n-- File downloaded to .%c%s\n\n", os.PathSeparator, filename)
}
