package utils

import "github.com/go-resty/resty/v2"

type RestyClientAdapter struct {
	Client *resty.Client
}

// NewRestyClientAdapter creates a new resty client adapter
func NewRestyClientApdater() *RestyClientAdapter {
	client := resty.New()
	return &RestyClientAdapter{
		Client: client,
	}
}

// NewRestyClientAdapterRequest creates a new resty client adapter request
func NewRestyClientApdaterRequest() *resty.Request {
	return resty.New().R()
}

// R returns the resty request
func (r *RestyClientAdapter) R() *resty.Request {
	return r.Client.R()
}
