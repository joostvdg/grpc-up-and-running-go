package internal

import (
	"context"
	"encoding/base64"
)

type BasicAuth struct {
	Username string
	Password string
}

func (b *BasicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	auth := b.Username + ":" + b.Password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))

	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (b *BasicAuth) RequireTransportSecurity() bool {
	return false // if we require tls with certs or not
}
