package auth_service

import (
	"fmt"
	"net/http"
	"time"
	"strings"
	"github.com/golang-jwt/jwt"
)


const (
    SigningKey = "DFSD,asdlfkjFSDFJRWFKDvkdlsLKjsde5tToDogge"
)


// Sign a JWT using the HS256 algorithm
// Generate the claims and pass them over here (claims like username(must) etc)
func signJWT(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	tokenString,err:=token.SignedString([]byte(SigningKey))
	if err!=nil{
		return "",err
	}
return tokenString,nil
}

// Verify a JWT using the HS256 algorithm
func verifyJWT(tokenString string) (jwt.MapClaims, int) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString,&claims, func(token *jwt.Token) (interface{}, error) {
		// Check the signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(SigningKey), nil
	})
	if err != nil {
		return nil, 401
	}
	if token.Valid {
		return claims, 200
	}
	return nil,401
}


// A handler function that signs a JWT and sends it to the client
func SignHandler(username string) (string,error){
	// Set the claims for the JWT
	claims := jwt.MapClaims{
		"sub": username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}
	// Sign the JWT
	token, err := signJWT(claims)
	if err != nil {
		return "",err
	}
	return token,nil

}


// A handler function that verifies a JWT sent by the client
func VerifyHandler(r *http.Request) (string,int){
	// Get the JWT from the request
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	tokenString := splitToken[1]
	// Verify the JWT
	claims, status := verifyJWT(tokenString)
	if status != 200 {
		return "",status
	}
	return claims["sub"].(string),200 

}
