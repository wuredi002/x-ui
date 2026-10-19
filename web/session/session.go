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
	// Store a value, matching the type registered with gob and read below.
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
		// Accept pointer-form sessions for compatibility with existing cookies.
		return user
	default:
		return nil
	}
}

func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != nil
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Options(sessions.Options{
		Path:   "/",
		MaxAge: -1,
	})
	s.Save()
}
