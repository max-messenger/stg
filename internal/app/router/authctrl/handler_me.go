package authctrl

import (
	"net/http"

	"stg/internal/app/router"
)

// meHandler godoc
//
// @Summary		get current user
// @Description	return user data from access_token
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param       Authorization  header    string  true  "Bearer token"
// @Success		200		{object} authResponse
// @Failure		401		{object} string
// @Router		/v1/auth/me [get]
// .
func (c *Controller) meHandler(w http.ResponseWriter, r *http.Request) {
	data, err := c.srv.ParseAndValidateToken(r.Context(), r.Header.Get("Authorization"), "access")
	if err != nil {
		http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)

		return
	}

	router.WriteResponse(w, http.StatusOK, meResponse{
		UserID:   data.User.ID,
		Username: data.User.Username,
		ChatID:   data.User.ChatID,
	})
}
