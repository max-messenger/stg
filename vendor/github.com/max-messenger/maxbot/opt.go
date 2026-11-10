package maxbot

import (
	"time"

	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
)

type Opt func(cli *Api)

func WithHTTPClient(cli maxbot.HttpClient) Opt {
	return func(b *Api) {
		b.conf.opts = append(b.conf.opts, maxbot.WithHTTPClient(cli))
	}
}

func WithBaseURL(baseURL string) Opt {
	return func(b *Api) {
		b.conf.opts = append(b.conf.opts, maxbot.WithBaseURL(baseURL))
	}
}

func WithPollingPause(d time.Duration) Opt {
	return func(b *Api) {
		b.conf.opts = append(b.conf.opts, maxbot.WithPollingPause(d))
	}
}

func WithPollingTimeout(d time.Duration) Opt {
	return func(b *Api) {
		b.conf.opts = append(b.conf.opts, maxbot.WithPollingTimeout(d))
	}
}

func WithErrorHandle(f func(error, Context)) Opt {
	return func(b *Api) {
		b.onError = f
	}
}

func WithWebhook(webhookUrl, secret string, types []string) Opt {
	return func(b *Api) {
		b.conf.webhookURL = webhookUrl
		b.conf.secret = secret
		b.conf.types = types
	}
}
