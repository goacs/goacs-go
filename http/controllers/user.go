package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/models/user"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
)

type UserUpdateRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password"`
}

type UserCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func UserCreate(ctx *gin.Context) {
	var request UserCreateRequest
	err := json.NewDecoder(ctx.Request.Body).Decode(&request)
	if err != nil {
		log.Println("Error in req")
	}

	userModel := user.User{
		Username: request.Username,
		Password: user.EncryptPassword(request.Password),
		Email:    request.Email,
	}

	userRepository := mysql.NewUserRepository(repository.GetConnection())
	userInstance, err := userRepository.CreateUser(&userModel)
	log.Print(userModel, userInstance)
	json.NewEncoder(ctx.Writer).Encode(userInstance)
}

func UserList(ctx *gin.Context) {
	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	userRepository := mysql.NewUserRepository(repository.GetConnection())
	users, total := userRepository.List(paginatorRequest)
	response.ResponsePaginatior(ctx, repository.NewPaginatorResponse(paginatorRequest, total, users))
}

func UserShow(ctx *gin.Context) {
	userRepository := mysql.NewUserRepository(repository.GetConnection())
	userModel, err := userRepository.Find(ctx.Param("uuid"))

	if err != nil {
		response.ResponseError(ctx, 404, "Not found", "")
		return
	}

	response.ResponseData(ctx, userModel)
}

func UserUpdate(ctx *gin.Context) {
	var updateRequest UserUpdateRequest
	_ = ctx.ShouldBindJSON(&updateRequest)

	validator := request.NewApiValidator(ctx, updateRequest)
	if err := validator.Validate(); err != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	userRepository := mysql.NewUserRepository(repository.GetConnection())
	userModel, err := userRepository.Find(ctx.Param("uuid"))

	if err != nil {
		response.ResponseError(ctx, 404, "Not found", "")
		return
	}

	userModel.Username = updateRequest.Username
	userModel.Email = updateRequest.Email

	hashedPassword := ""
	if updateRequest.Password != "" {
		hashedPassword = user.EncryptPassword(updateRequest.Password)
	}

	if err := userRepository.Update(&userModel, hashedPassword); err != nil {
		response.Response500(ctx, "Cannot update user", err)
		return
	}

	response.ResponseData(ctx, userModel)
}

func UserDelete(ctx *gin.Context) {
	targetUUID := ctx.Param("uuid")

	if targetUUID == CurrentUserUUID(ctx) {
		response.ResponseError(ctx, 422, "Cannot delete your own account", "")
		return
	}

	userRepository := mysql.NewUserRepository(repository.GetConnection())
	if err := userRepository.Delete(targetUUID); err != nil {
		response.Response500(ctx, "Cannot delete user", err)
		return
	}

	response.ResponseData(ctx, "")
}
