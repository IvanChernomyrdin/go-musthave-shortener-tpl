package middleware

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config"
	"github.com/google/uuid"
)

var encryptionKey = []byte(config.EncryptionKey)

func CookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string
		//получаем куки
		cookie, err := r.Cookie("userID")
		// если их нет создаём новый uuid и куки
		if err != nil {
			userID = uuid.New().String()
			setEncryptedCookie(w, userID)
		} else {
			userID, err = decrypt(cookie.Value)
			if err != nil {
				userID = uuid.New().String()
				setEncryptedCookie(w, userID)
			}
		}
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func setEncryptedCookie(w http.ResponseWriter, userID string) {
	encrypted, err := encrypt(userID)
	if err != nil {
		encrypted = userID
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "userID",
		Value:    encrypted,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func encrypt(userID string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(userID), nil)
	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func decrypt(val string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(val)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext to short")
	}
	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plainText), err
}
