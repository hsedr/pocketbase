package pocketbase

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/r--w/pocketbase/internal"
	"golang.org/x/sync/singleflight"
)

type authorizeToken struct {
	client      *resty.Client
	url         string
	token       string
	model       map[string]interface{}
	tokenValid  time.Time
	tokenSingle singleflight.Group
}

func newAuthorizeToken(c *resty.Client, url string, token string) authStore {
	c.SetHeader("Authorization", token)
	return &authorizeToken{
		client:      c,
		url:         url,
		token:       token,
		tokenSingle: singleflight.Group{},
	}
}

func (a *authorizeToken) authorize() error {
	type authResponse struct {
		Token  string                 `json:"token"`
		Record map[string]interface{} `json:"record"`
		Admin  map[string]interface{} `json:"admin"`
	}
	_, err, _ := a.tokenSingle.Do("auth-refresh", func() (interface{}, error) {
		if time.Now().Before(a.tokenValid) {
			return nil, nil
		}
		resp, err := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", a.token).
			SetResult(&authResponse{}).
			Post(a.url)
		if err != nil {
			return nil, fmt.Errorf("[auth-refresh] can't send request to pocketbase %w", err)
		}
		if resp.IsError() {
			return nil, fmt.Errorf("[auth-refresh] pocketbase returned status: %d, msg: %s, err %w",
				resp.StatusCode(),
				resp.String(),
				ErrInvalidResponse,
			)
		}
		auth := *resp.Result().(*authResponse)
		a.token = auth.Token
		a.model = internal.First(len(auth.Admin) > 0, auth.Admin, auth.Record)
		a.client.SetHeader("Authorization", auth.Token)
		a.tokenValid = time.Now().Add(60 * time.Minute)
		return nil, nil
	})
	return err
}

func (a *authorizeToken) IsValid() bool {
	return time.Now().Before(a.tokenValid)
}

func (a *authorizeToken) Token() string {
	return a.token
}

func (a *authorizeToken) Model(o interface{}) error {
	return mapstructure.Decode(a.model, o)
}
