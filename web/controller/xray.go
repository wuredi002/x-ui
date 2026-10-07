package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type XrayController struct {
	BaseController
}

func NewXrayController(g *gin.RouterGroup) *XrayController {
	a := &XrayController{}
	a.initRouter(g)
	return a
}

func (a *XrayController) initRouter(g *gin.RouterGroup) {
	panel := g.Group("/xray")
	panel.Use(a.checkLogin)

	panel.GET("/", a.index)
	panel.GET("/inbounds", a.inbounds)
	panel.GET("/setting", a.setting)

	NewInboundController(panel)
	NewSettingController(panel)
	legacy := g.Group("/xui")
	legacy.Use(a.checkLogin)
	legacy.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path")+"xray/") })
	legacy.GET("/inbounds", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path")+"xray/inbounds")
	})
	legacy.GET("/setting", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path")+"xray/setting")
	})
}

func (a *XrayController) index(c *gin.Context) {
	html(c, "index.html", "系统状态", nil)
}

func (a *XrayController) inbounds(c *gin.Context) {
	html(c, "inbounds.html", "入站列表", nil)
}

func (a *XrayController) setting(c *gin.Context) {
	html(c, "setting.html", "设置", nil)
}
