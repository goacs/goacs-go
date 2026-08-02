package controllers

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/lib"
	"goacs/models/user"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
	"time"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	User  user.User `json:"user"`
	Token string    `json:"token"`
}

func Login(ctx *gin.Context) {
	var loginRequest LoginRequest

	err := json.NewDecoder(ctx.Request.Body).Decode(&loginRequest)

	if err != nil {
		log.Println("Error in req ", err)
	}

	validator := request.NewApiValidator(ctx, loginRequest)
	verr := validator.Validate()

	if verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	userRepository := mysql.NewUserRepository(repository.GetConnection())
	userByAuthData, err := userRepository.GetUserByAuthData(loginRequest.Username, loginRequest.Password)

	if err != nil {
		log.Println("Cannot find userByAuthData", err.Error())
		response.ResponseError(ctx, 404, "Cannot find userByAuthData", err)
		return
	}

	loginResponse := LoginResponse{
		User:  userByAuthData,
		Token: NewTokenForUser(userByAuthData),
	}

	response.ResponseData(ctx, loginResponse)
}

// Logout is a no-op on the server: JWTs here are stateless (no blacklist), so
// the client simply discards the token. Kept as a real endpoint so the
// frontend has a single, uniform auth contract to call on sign-out.
func Logout(ctx *gin.Context) {
	response.ResponseData(ctx, "")
}

func Refresh(ctx *gin.Context) {
	userRepository := mysql.NewUserRepository(repository.GetConnection())
	userModel, err := userRepository.Find(CurrentUserUUID(ctx))

	if err != nil {
		response.ResponseError(ctx, 404, "Cannot find user", err)
		return
	}

	response.ResponseData(ctx, LoginResponse{
		User:  userModel,
		Token: NewTokenForUser(userModel),
	})
}

func CurrentUserUUID(ctx *gin.Context) string {
	value, exists := ctx.Get("user_uuid")
	if !exists {
		return ""
	}

	uuid, _ := value.(string)
	return uuid
}

func NewTokenForUser(user user.User) string {
	env := new(lib.Env)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Minute * 120).Unix(),
		Subject:   user.Uuid,
		Issuer:    "user",
	})

	tokenString, err := token.SignedString([]byte(env.Get("JWT_SECRET", "")))
	if err != nil {
		log.Println("Error while generating token ", err)
	}
	return tokenString
}
