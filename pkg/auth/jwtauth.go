package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func GenerateToken(userId int, key string, expiry int64) (string, int64, error) {
	jwtKey := []byte(key)
	tokenExpirationTime := time.Now().Add(time.Duration(expiry) * time.Minute).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": userId,
		"exp":    tokenExpirationTime,
	})

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", 0, err
	}

	return tokenString, tokenExpirationTime, nil
}

func VerifyToken(tokenString string, key string) (uint, error) {
	jwtKey := []byte(key)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtKey, nil
	})

	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	userIdFloat, ok := claims["userId"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user ID format")
	}
	userId := uint(userIdFloat)

	return userId, nil
}
