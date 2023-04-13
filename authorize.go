package pocketbase

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hsedr/pocketbase/internal"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/sync/singleflight"
)

type authStore interface {
	authorizer
	IsValid() bool
	Token() string
	Model(interface{}) error
}

type authorizer interface {
	authorize() error
}

type authorizeNoOp struct{}

func (a authorizeNoOp) authorize() error {
	return nil
}

func (a authorizeNoOp) IsValid() bool {
	return false
}

func (a authorizeNoOp) Token() string {
	return ""
}

func (a authorizeNoOp) Model(o interface{}) error {
	return nil
}

type authorizeEmailPassword struct {
	email       string
	password    string
	token       string
	model       map[string]interface{}
	tokenValid  time.Time
	client      *resty.Client
	url         string
	tokenSingle singleflight.Group
}

func newAuthorizeEmailPassword(c *resty.Client, url string, email string, password string) authStore {
	return &authorizeEmailPassword{
		client:      c,
		email:       email,
		password:    password,
		url:         url,
		tokenSingle: singleflight.Group{},
	}
}

func (a *authorizeEmailPassword) authorize() error {
	type authResponse struct {
		Token  string                 `json:"token"`
		Record map[string]interface{} `json:"record"`
		Admin  map[string]interface{} `json:"admin"`
	}

	_, err, _ := a.tokenSingle.Do("auth", func() (interface{}, error) {
		if time.Now().Before(a.tokenValid) {
			return nil, nil
		}

		resp, err := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]interface{}{
				"identity": a.email,
				"password": a.password,
			}).
			SetResult(&authResponse{}).
			SetHeader("Authorization", "").
			Post(a.url)

		if err != nil {
			return nil, fmt.Errorf("[auth] can't send request to pocketbase %w", err)
		}

		if resp.IsError() {
			return nil, fmt.Errorf("[auth] pocketbase returned status: %d, msg: %s, err %w",
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

func (a *authorizeEmailPassword) IsValid() bool {
	return time.Now().Before(a.tokenValid)
}

func (a *authorizeEmailPassword) Token() string {
	return a.token
}

func (a *authorizeEmailPassword) Model(o interface{}) error {
	return mapstructure.Decode(a.model, o)
}
