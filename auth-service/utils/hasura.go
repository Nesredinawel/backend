package utils

import (
	"auth-service/models"
)

var GlobalHasura *HasuraClient

func InitHasura(cfg Config) {
	GlobalHasura = NewHasuraClient(cfg)
}

func UpsertUserInHasura(cfg Config, user models.User) (string, *ServiceError) {
	return GlobalHasura.UpsertUser(user)
}

func UpdateUserPassword(cfg Config, userID, password string) (string, *ServiceError) {
	return GlobalHasura.UpdatePassword(userID, password)
}

func GetUserByID(cfg Config, userID string) (models.User, *ServiceError) {
	return GlobalHasura.GetUserByID(userID)
}

func GetUserByEmail(cfg Config, email string) (models.User, *ServiceError) {
	return GlobalHasura.GetUserByEmail(email)
}

func CreateEmptyUserProfile(cfg Config, userID string) *ServiceError {
	return GlobalHasura.CreateEmptyUserProfile(userID)
}

func GetUserProfileFromHasura(cfg Config, userID string) (*models.UserProfile, *ServiceError) {
	return GlobalHasura.GetUserProfile(userID)
}

func UpdateUserProfileInHasura(cfg Config, userID string, input models.UserProfileInput) (*models.UserProfile, *ServiceError) {
	return GlobalHasura.UpdateUserProfile(userID, input)
}
