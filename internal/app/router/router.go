package router

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"stg/internal/app/services/auth"
	"stg/pkg/http/middlewares"
)

// Controller api controller.
type Controller interface {
	Register(r chi.Router)
}

// AuthMiddleware проверяет access_token куку и кладёт userID в контекст.
type AuthMiddleware func(http.Handler) http.Handler

type Controllers []Controller

// 	@title          STG API
// 	@version        1.0
// 	@description    This is a stg API.

//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func NewRouter(logger *zap.Logger, controllers Controllers, corsCfg middlewares.CORSConfig) (http.Handler, error) {
	router := chi.NewRouter()

	router.Use(middlewares.CORS(corsCfg))

	router.Route("/", func(r chi.Router) {
		r.Use(
			middlewares.Metrics(),
			middlewares.AccessLog(logger),
			middlewares.Trace(),
		)
		for _, ctrl := range controllers {
			ctrl.Register(r)
		}
	})

	return router, nil
}

func NewAuthMiddleware(svc *auth.Service) AuthMiddleware {
	return middlewares.AuthMiddleware(func(ctx context.Context, token string) (int64, error) {
		claims, err := svc.ParseAndValidateToken(ctx, token, "access")
		if err != nil {
			return 0, err
		}

		return claims.User.ID, nil
	})
}
