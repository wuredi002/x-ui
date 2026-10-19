package controller

import (
	"fmt"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"strconv"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/web/global"
	"x-ui/web/service"
	"x-ui/web/session"
)

type InboundController struct {
	inboundService service.InboundService
	xrayService    service.XrayService
}

func NewInboundController(g *gin.RouterGroup) *InboundController {
	a := &InboundController{}
	a.initRouter(g)
	a.startTask()
	return a
}

func (a *InboundController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/inbound")

	g.POST("/list", a.getInbounds)
	g.POST("/detect-remark", a.detectRemark)
	g.POST("/add", a.addInbound)
	g.POST("/del/:id", a.delInbound)
	g.POST("/update/:id", a.updateInbound)
}

func (a *InboundController) startTask() {
	webServer := global.GetWebServer()
	c := webServer.GetCron()
	c.AddFunc("@every 10s", func() {
		if a.xrayService.IsNeedRestartAndSetFalse() {
			err := a.xrayService.RestartXray(false)
			if err != nil {
				logger.Error("restart xray failed:", err)
			}
		}
	})
}

// detectRemark identifies this panel server's public IPv4 and its country.
func (a *InboundController) detectRemark(c *gin.Context) {
	client := &http.Client{Timeout: 4 * time.Second}
	providers := []string{
		"https://test.ipw.cn",
		"https://icanhazip.com",
		"https://api.ipify.org",
		"https://ident.me",
		"https://ipecho.net/plain",
		"https://checkip.amazonaws.com",
		"https://bot.whatismyipaddress.com",
		"https://ipinfo.io/ip",
		"https://myexternalip.com/raw",
		"https://ifconfig.me",
		"https://wgetip.com",
		"https://ip.seeip.org",
	}
	var publicIP string
	for _, endpoint := range providers {
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		if readErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		candidate := strings.TrimSpace(string(body))
		ip := net.ParseIP(candidate)
		if ip != nil && ip.To4() != nil {
			publicIP = ip.To4().String()
			break
		}
	}
	if publicIP == "" {
		jsonMsg(c, "识别公网 IPv4", fmt.Errorf("所有 IPv4 查询服务均未返回有效地址"))
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet,
		"http://ip-api.com/json/"+publicIP+"?lang=zh-CN&fields=status,country,message", nil)
	if err != nil {
		jsonMsg(c, "识别国家", err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		jsonMsg(c, "识别国家", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		jsonMsg(c, "识别国家", fmt.Errorf("ip-api.com 返回 HTTP %d", resp.StatusCode))
		return
	}
	var result struct {
		Status  string `json:"status"`
		Country string `json:"country"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&result); err != nil {
		jsonMsg(c, "识别国家", err)
		return
	}
	if result.Status != "success" || strings.TrimSpace(result.Country) == "" {
		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = "未返回国家信息"
		}
		jsonMsg(c, "识别国家", fmt.Errorf("%s", message))
		return
	}
	jsonObj(c, gin.H{
		"country": result.Country,
		"ip":      publicIP,
		"remark":  result.Country + "-" + publicIP + "-网路跳越",
	}, nil)
}

func (a *InboundController) getInbounds(c *gin.Context) {
	user := session.GetLoginUser(c)
	inbounds, err := a.inboundService.GetInbounds(user.Id)
	if err != nil {
		jsonMsg(c, "获取", err)
		return
	}
	jsonObj(c, inbounds, nil)
}

func (a *InboundController) addInbound(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}
	user := session.GetLoginUser(c)
	inbound.UserId = user.Id
	inbound.Enable = true
	inbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)
	err = a.inboundService.AddInbound(inbound)
	jsonMsg(c, "添加", err)
	if err == nil {
		a.xrayService.SetToNeedRestart()
	}
}

func (a *InboundController) delInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "删除", err)
		return
	}
	err = a.inboundService.DelInbound(id)
	jsonMsg(c, "删除", err)
	if err == nil {
		a.xrayService.SetToNeedRestart()
	}
}

func (a *InboundController) updateInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "修改", err)
		return
	}
	inbound := &model.Inbound{
		Id: id,
	}
	err = c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, "修改", err)
		return
	}
	err = a.inboundService.UpdateInbound(inbound)
	jsonMsg(c, "修改", err)
	if err == nil {
		a.xrayService.SetToNeedRestart()
	}
}
