// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package verifyapi implements the exchange of the verification code
// (short term token) for a long term token that can be used to get a
// verification certification to send to the key server.
//
// This is steps 4/5 as specified here:
// https://developers.google.com/android/exposure-notifications/verification-system#flow-diagram
package verifyapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/exposure-notifications-verification-server/pkg/api"
	"github.com/google/exposure-notifications-verification-server/pkg/config"
	"github.com/google/exposure-notifications-verification-server/pkg/controller"
	"github.com/google/exposure-notifications-verification-server/pkg/database"
	"github.com/google/exposure-notifications-verification-server/pkg/jwthelper"
	"github.com/google/exposure-notifications-verification-server/pkg/logging"
	"github.com/google/exposure-notifications-verification-server/pkg/signer"

	"github.com/google/exposure-notifications-server/pkg/cache"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VerifyAPI is a controller for the verification code verification API.
type VerifyAPI struct {
	config   *config.Config
	db       *database.Database
	logger   *zap.SugaredLogger
	keyCache *cache.Cache
	signer   signer.KeyManager
}

func New(ctx context.Context, config *config.Config, db *database.Database, keyCache *cache.Cache, signer signer.KeyManager) controller.Controller {
	return &VerifyAPI{config, db, logging.FromContext(ctx), keyCache, signer}
}

func (v *VerifyAPI) Execute(c *gin.Context) {
	// this is not an authenticated API w/ session cookie. Uses API Key instead.
	var request api.VerifyCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		v.logger.Errorf("failed to bind request: %v", err)
		c.JSON(http.StatusOK, api.VerifyCodeResponse{Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Load the authorized app by API key using the write thru cache.
	authAppCache, err := v.keyCache.WriteThruLookup(request.APIKey,
		func() (interface{}, error) {
			aa, err := v.db.FindAuthoirizedAppByAPIKey(request.APIKey)
			if err != nil {
				return nil, err
			}
			return aa, nil
		})
	authApp, ok := authAppCache.(*database.AuthorizedApp)
	if !ok {
		authApp = nil
	}

	// Check if the API key is authorized.
	if err != nil || authApp == nil || authApp.DeletedAt != nil {
		v.logger.Errorf("unauthorized API Key: %v err: %v", request.APIKey, err)
		c.JSON(http.StatusUnauthorized, api.VerifyCodeResponse{Error: "unauthorized: API Key invalid"})
		return
	}

	// Get the signer based on Key configuration.
	signer, err := v.signer.NewSigner(c.Request.Context(), v.config.TokenSigningKey)
	if err != nil {
		v.logger.Errorf("unable to get signing key: %v", err)
		c.JSON(http.StatusInternalServerError, api.VerifyCodeResponse{Error: "internal server error - unable to sign tokens"})
		return
	}

	// Exchange the short term verification code for a long term verification token.
	// The token can be used to sign TEKs later.
	verificationToken, err := v.db.IssueToken(request.VerificationCode, v.config.VerificationTokenDuration)
	if err != nil {
		v.logger.Errorf("error issuing verification token: %v", err)
		if errors.Is(err, database.ErrVerificationCodeExpired) || errors.Is(err, database.ErrVerificationCodeUsed) {
			c.JSON(http.StatusBadRequest, api.VerifyCodeResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, api.VerifyCodeResponse{Error: "internal serer error"})
		return
	}

	subject := verificationToken.TestType + "." + verificationToken.FormatTestDate()
	now := time.Now().UTC()
	claims := &jwt.StandardClaims{
		Audience:  v.config.TokenIssuer,
		ExpiresAt: now.Add(v.config.VerificationTokenDuration).Unix(),
		Id:        verificationToken.TokenID,
		IssuedAt:  now.Unix(),
		Issuer:    v.config.TokenIssuer,
		Subject:   subject,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedJWT, err := jwthelper.SignJWT(token, signer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.VerifyCodeResponse{Error: "error signing token, must obtain new verification code"})
		return
	}

	c.JSON(http.StatusOK, api.VerifyCodeResponse{
		TestType:          verificationToken.TestType,
		TestDate:          verificationToken.FormatTestDate(),
		VerificationToken: signedJWT,
	})
}