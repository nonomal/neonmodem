package hyperuplink

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mrusme/neonmodem/system/hyperuplink/api"
	"github.com/mrusme/neonmodem/system/prompt"
)

func (sys *System) Connect(sysURL string) error {
	username, err := prompt.Line("Please enter your username", "username")
	if err != nil {
		return err
	}

	secret, err := prompt.Secret("Please enter your API token", "API token")
	if err != nil {
		return err
	}

	token := strings.TrimSpace(secret)
	if token == "" {
		return errors.New("no API token was entered")
	}

	clientCfg := api.NewDefaultClientConfig(sysURL, "", token, sys.logger)
	client := api.NewClient(&clientCfg)
	session, err := client.Session.Whoami(context.Background())
	if err != nil {
		if errors.Is(err, api.ErrNotAnAPI) {
			return fmt.Errorf(
				"%s doesn't look quite right: %w. Please make sure you're "+
					"using the address of the Hyperuplink API, which listens on a "+
					"different port than the web interface, port 3001 by default",
				sysURL, err,
			)
		}
		return fmt.Errorf(
			"could not authenticate against %s: %w", sysURL, err)
	}
	if session.User.Username != username {
		fmt.Printf(
			"Note: this token belongs to '%s', not '%s'; using '%s'.\n",
			session.User.Username, username, session.User.Username,
		)
		username = session.User.Username
	}

	credentials := make(map[string]string)
	credentials["username"] = username
	credentials["token"] = token

	if sys.config == nil {
		sys.config = make(map[string]interface{})
	}
	sys.config["url"] = sysURL
	sys.config["credentials"] = credentials

	return nil
}
