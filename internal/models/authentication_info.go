package models

type AuthenticationInfo struct {
	User *User `json:"user"`
}

func NewAuthenticationInfo(
	user *User,
) *AuthenticationInfo {
	return &AuthenticationInfo{
		User: user,
	}
}
