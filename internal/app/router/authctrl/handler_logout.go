package authctrl

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"stg/internal/app/router"
)

// logoutHandler godoc
//
// @Summary		logout
// @Description	invalidate refresh token and clear auth
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Success		200		{object} string
// @Failure		401		{object} string
// @Failure		500		{object} string
// @Router		/v1/auth/logout [post]
// .
func (c *Controller) logoutHandler(w http.ResponseWriter, r *http.Request) {
	err := c.srv.Logout(context.Background(), r.Header.Get("Authorization"))
	if err != nil {
		c.logger.Debug("logout failed", zap.Error(err))
		router.WriteError(w, http.StatusUnauthorized, errors.New("invalid refresh token"))

		return
	}

	router.WriteSuccess(w, nil)
}
