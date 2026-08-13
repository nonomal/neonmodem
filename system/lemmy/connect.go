package lemmy

import (
	"github.com/mrusme/neonmodem/system/prompt"
)

func (sys *System) Connect(sysURL string) error {
	// Request input from user
	username, err := prompt.Line("Please enter your username or email", "username")
	if err != nil {
		return err
	}

	// Request input from user
	password, err := prompt.Secret("Please enter your password", "password")
	if err != nil {
		return err
	}

	// Credentials
	credentials := make(map[string]string)
	credentials["username"] = username
	credentials["password"] = password

	if sys.config == nil {
		sys.config = make(map[string]interface{})
	}
	sys.config["url"] = sysURL
	sys.config["credentials"] = credentials

	return nil
}
