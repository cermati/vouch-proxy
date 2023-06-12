/*

Copyright 2020 The Vouch Proxy Authors.
Use of this source code is governed by The MIT License (MIT) that
can be found in the LICENSE file. Software distributed under The
MIT License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
OR CONDITIONS OF ANY KIND, either express or implied.

*/

package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/vouch/vouch-proxy/pkg/cfg"
	"github.com/vouch/vouch-proxy/pkg/domains"
	"github.com/vouch/vouch-proxy/pkg/structs"
)

var log *zap.SugaredLogger

// Configure see main.go configure()
func Configure() {
	log = cfg.Logging.Logger
}

// PrepareTokensAndClient setup the client, usually for a UserInfo request
func PrepareTokensAndClient(r *http.Request, ptokens *structs.PTokens, setProviderToken bool, opts ...oauth2.AuthCodeOption) (*http.Client, *oauth2.Token, error) {
	// Avoid modifying global variable, copy the value of global OauthClient to local variable
	oauthClient := *cfg.OAuthClient
	if oauthClient.Scopes != nil {
		// clone array content
		oauthClient.Scopes = append([]string{}, cfg.OAuthClient.Scopes...)
	}

	if len(cfg.GenOAuth.RedirectURLs) > 0 {
		found := false
		domain := domains.Matches(r.Host)
		for _, v := range cfg.GenOAuth.RedirectURLs {
			if strings.Contains(v, domain) {
				found = true
				oauthClient.RedirectURL = v
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("no callback_url matched %s (is the `Host` header being passed to Vouch Proxy?)", domain)
		}
	}

	providerToken, err := oauthClient.Exchange(r.Context(), r.URL.Query().Get("code"), opts...)
	if err != nil {
		return nil, nil, err
	}
	ptokens.PAccessToken = providerToken.AccessToken

	if setProviderToken {
		if providerToken.Extra("id_token") != nil {
			// Certain providers (eg. gitea) don't provide an id_token
			// and it's not necessary for the authentication phase
			ptokens.PIdToken = providerToken.Extra("id_token").(string)
		} else {
			log.Debugf("id_token missing - may not be supported by this provider")
		}
	}

	log.Debugf("ptokens: accessToken length: %d, IdToken length: %d", len(ptokens.PAccessToken), len(ptokens.PIdToken))
	client := oauthClient.Client(r.Context(), providerToken)
	return client, providerToken, err
}

// MapClaims populate CustomClaims from userInfo for each configure claims header
func MapClaims(claims []byte, customClaims *structs.CustomClaims) error {
	var f interface{}
	err := json.Unmarshal(claims, &f)
	if err != nil {
		log.Error("Error unmarshaling claims")
		return err
	}
	m := f.(map[string]interface{})
	for k := range m {
		var found = false
		for claim := range cfg.Cfg.Headers.ClaimsCleaned {
			if k == claim {
				found = true
			}
		}
		if found == false {
			delete(m, k)
		}
	}
	customClaims.Claims = m
	return nil
}
