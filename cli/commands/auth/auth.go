package auth

import (
	"log"
	"yeetfile/cli/globals"
)

func IsUserAuthenticated() (bool, error) {
	_, err := globals.API.GetSession()
	if err != nil {
		if globals.Config.DebugMode {
			log.Printf("Server returned session error: %v\n", err)
		}

		// Ensure keys are removed if the user has an older session
		if len(globals.API.Session) > 0 {
			resetErr := globals.Config.Reset()
			if resetErr != nil {
				return false, resetErr
			}
		}

		return false, err
	}

	return true, nil
}
