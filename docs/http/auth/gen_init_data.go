// Генератор тестового init_data для MAX мини-приложения.
// Запуск: go run docs/http/auth/gen_init_data.go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

func main() {
	const botToken = "test-token" //nolint:gosec // тестовый токен для локальной генерации

	authDate := time.Now().Unix()

	values := url.Values{
		"query_id":    {"AAHdF3e7"},
		"user":        {`{"id":12345,"first_name":"Max","last_name":"User","username":"maxuser","language_code":"ru"}`},
		"chat":        {`{"id":67890,"type":"DIALOG"}`},
		"auth_date":   {fmt.Sprintf("%d", authDate)},
		"start_param": {"promo"},
	}

	// Вычисляем hash по URL-декодированным значениям (как MAX)
	pairs := make([]string, 0, len(values))
	for key := range values {
		pairs = append(pairs, key+"="+values.Get(key))
	}

	sort.Strings(pairs)

	launchParams := strings.Join(pairs, "\n")

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	sig := hmac.New(sha256.New, secretKey.Sum(nil))
	sig.Write([]byte(launchParams))
	hash := hex.EncodeToString(sig.Sum(nil))

	values.Set("hash", hash)

	fmt.Printf("{\n  \"init_data\": %q\n}\n", values.Encode()) //nolint
}
