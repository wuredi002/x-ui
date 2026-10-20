package session

import (
	"encoding/gob"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"x-ui/database/model"
)

const (
	loginUser = "LOGIN_USER"
)

func init() {
	gob.Register(model.User{})
}

func SetLoginUser(c *gin.Context, user *model.User) error {
	s := sessions.Default(c)
	if user == nil {
		s.Delete(loginUser)
		return s.Save()
	}
	// Store the registered value type so it matches gob registration and the reader.
	s.Set(loginUser, *user)
	return s.Save()
}

func GetLoginUser(c *gin.Context) *model.User {
	s := sessions.Default(c)
	obj := s.Get(loginUser)
	switch user := obj.(type) {
	case model.User:
		return &user
	case *model.User:
		// Support sessions created by older code that stored a pointer.
		return user
	default:
		return nil
	}
}

func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != nil
}

func ClearSession(c *gin.Context) error {
	s := sessions.Default(c)
	s.Clear()
	s.Options(sessions.Options{
		Path:   "/",
		MaxAge: -1,
	})
	return s.Save()
}
