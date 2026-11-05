package authctrl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"stg/internal/app/router"
)

type tokenRequest struct {
	InitData string `json:"init_data"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type meResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	ChatID   int64  `json:"chat_id"`
}

// loginHandler godoc
//
// @Summary		authenticate user
// @Description	validate MAX init_data and issue access/refresh token pair
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param       request body     tokenRequest true "request body"
// @Success		200		{object} authResponse
// @Failure		400		{object} string
// @Failure		401		{object} string
// @Failure		500		{object} string
// @Router		/v1/auth/token [post]
// .
func (c *Controller) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		router.WriteError(w, http.StatusBadRequest, errors.New("invalid request body"))

		return
	}

	if req.InitData == "" {
		router.WriteError(w, http.StatusBadRequest, errors.New("init_data is required"))

		return
	}

	data, err := c.srv.Authenticate(context.Background(), req.InitData)
	if err != nil {
		router.WriteError(w, http.StatusUnauthorized, errors.New("invalid init data"))

		return
	}

	router.WriteSuccess(w, authResponse{
		AccessToken:  data.TokenPair.AccessToken,
		RefreshToken: data.TokenPair.RefreshToken,
	})
}
