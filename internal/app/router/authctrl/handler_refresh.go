package authctrl

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"stg/internal/app/router"
)

// refreshHandler godoc
//
// @Summary		refresh tokens
// @Description	rotate access/refresh token pair using refresh_token
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Success		200		{object} authResponse
// @Failure		401		{object} string
// @Failure		500		{object} string
// @Router		/v1/auth/refresh [post]
// .
func (c *Controller) refreshHandler(w http.ResponseWriter, r *http.Request) {
	data, err := c.srv.Refresh(context.Background(), r.Header.Get("Authorization"))
	if err != nil {
		c.logger.Debug("refresh token failed", zap.Error(err))
		router.WriteError(w, http.StatusUnauthorized, errors.New("invalid refresh token"))

		return
	}

	router.WriteSuccess(w, authResponse{
		AccessToken:  data.TokenPair.AccessToken,
		RefreshToken: data.TokenPair.RefreshToken,
	})
}
