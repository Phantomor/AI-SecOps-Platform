// internal/pkg/jwt.go
package pkg

import (
	"errors"
	"time"

	"sentinel-agent-go/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 自定义载荷，可以存放用户ID和角色
type CustomClaims struct {
	UserID      uint   `json:"user_id"`
	UserAccount string `json:"user_account"`
	UserRole    string `json:"user_role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT
func GenerateToken(userID uint, userAccount, userRole string) (string, error) {
	claims := CustomClaims{
		UserID:      userID,
		UserAccount: userAccount,
		UserRole:    userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			// 设置过期时间，比如 24 小时
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "sentinel-agent-go",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Global.Server.JwtSecret))
}

// ParseToken 解析 JWT
func ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Global.Server.JwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
