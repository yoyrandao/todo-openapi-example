package openid

import (
	"encoding/json"
	"net/http"
)

type WellKnownConfiguration struct {
	TokenEndpoint string `json:"token_endpoint"`
	JWKSUri       string `json:"jwks_uri"`
}

func NewWellKnownConfiguration(wellKnownEndpoint string) (*WellKnownConfiguration, error) {
	request, err := http.NewRequest("GET", wellKnownEndpoint, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	var config WellKnownConfiguration
	if err := json.NewDecoder(response.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
