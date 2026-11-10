package middleware

import (
	"errors"
	"log"

	"github.com/max-messenger/maxbot"
)

type RecoverFunc = func(error, maxbot.Context)

func Recover(onError ...RecoverFunc) maxbot.MiddlewareFunc {
	return func(next maxbot.HandlerFunc) maxbot.HandlerFunc {
		return func(c maxbot.Context) error {
			f := func(err error, context maxbot.Context) {
				if c != nil {
					log.Println(c.Update().ChatID, err)
					return
				}

				log.Println(err)
			}
			if len(onError) > 0 {
				f = onError[0]
			}
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						f(err, c)
					} else if s, ok := r.(string); ok {
						f(errors.New(s), c)
					}
				}
			}()

			return next(c)
		}
	}
}
