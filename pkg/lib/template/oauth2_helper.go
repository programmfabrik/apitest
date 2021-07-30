package template

import (
	"encoding/json"

	"golang.org/x/oauth2"
)

type oAuth2TokenExtended struct {
	*oauth2.Token
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// ReadOAuthReturnValue checks the return values from an OAUTH client and
// stores the error in an extended struct of the oAuth token
func readOAuthReturnValue(t *oauth2.Token, err error) (tE oAuth2TokenExtended) {
	if t == nil {
		t = &oauth2.Token{} // Make sure we have "AccessToken" in our struct
	}
	tE = oAuth2TokenExtended{Token: t}
	if err != nil {
		switch v := err.(type) {
		case *oauth2.RetrieveError:
			err = json.Unmarshal(v.Body, &tE)
			if err != nil {
				tE.Error = err.Error()
				tE.ErrorDescription = string(v.Body)
				err = nil
			}
		}
	}
	return tE
}
