package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xray/database/model"
	"xray/web/service"
	"xray/web/session"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

func (a *XrayController) getClashSubscriptionURL(c *gin.Context) {
	user := session.GetLoginUser(c)
	secret, err := (&service.SettingService{}).GetSecret()
	if err != nil {
		jsonMsg(c, "生成 Clash 订阅地址", err)
		return
	}
	token := clashSubscriptionToken(secret, user.Id)
	jsonObj(c, c.GetString("base_path")+"sub/"+strconv.Itoa(user.Id)+"/"+token, nil)
}

func (a *XrayController) clashSubscription(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil || userID <= 0 {
		c.Status(http.StatusNotFound)
		return
	}
	secret, err := (&service.SettingService{}).GetSecret()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	expected := clashSubscriptionToken(secret, userID)
	provided, err := hex.DecodeString(c.Param("token"))
	want, _ := hex.DecodeString(expected)
	if err != nil || !hmac.Equal(provided, want) {
		c.Status(http.StatusNotFound)
		return
	}

	inbounds, err := (&service.InboundService{}).GetInbounds(userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load subscription")
		return
	}
	proxies := make([]map[string]interface{}, 0)
	host := c.Request.Host
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	} else {
		host = strings.Trim(host, "[]")
	}
	country, publicIP := detectCountryAndPublicIPv4()
	now := time.Now().UnixMilli()
	for _, inbound := range inbounds {
		if !inbound.Enable || (inbound.ExpiryTime > 0 && inbound.ExpiryTime <= now) ||
			(inbound.Total > 0 && inbound.Up+inbound.Down >= inbound.Total) {
			continue
		}
		proxies = append(proxies, inboundClashProxies(inbound, host)...)
	}
	if len(proxies) == 0 {
		c.String(http.StatusNotFound, "no available proxies")
		return
	}
	decorateProxyNames(proxies, country, publicIP)
	makeProxyNamesUnique(proxies)
	config := map[string]interface{}{
		"mixed-port": 7890,
		"allow-lan":  false,
		"mode":       "rule",
		"log-level":  "info",
		"ipv6":       false,
		"proxies":    proxies,
		"proxy-groups": []map[string]interface{}{
			{"name": "代理选择", "type": "select", "proxies": []string{"自动选择", "DIRECT"}},
			{"name": "自动选择", "type": "url-test", "url": "https://www.gstatic.com/generate_204", "interval": 300, "proxies": proxyNames(proxies)},
		},
		"rules": []string{"MATCH,代理选择"},
	}
	body, err := yaml.Marshal(config)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to encode subscription")
		return
	}
	c.Header("Content-Disposition", `inline; filename="clash-meta.yaml"`)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/yaml; charset=utf-8", body)
}

func decorateProxyNames(proxies []map[string]interface{}, country, publicIP string) {
	prefix := countryIPPrefix(country, publicIP)
	for _, proxy := range proxies {
		if name, ok := proxy["name"].(string); ok {
			proxy["name"] = prefix + "-" + name
		}
	}
}

func clashSubscriptionToken(secret []byte, userID int) string {
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "clash-subscription:%d", userID)
	return hex.EncodeToString(mac.Sum(nil))
}

func proxyNames(proxies []map[string]interface{}) []string {
	names := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		if name, ok := proxy["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

func makeProxyNamesUnique(proxies []map[string]interface{}) {
	used := make(map[string]bool, len(proxies))
	for _, proxy := range proxies {
		name, _ := proxy["name"].(string)
		candidate := name
		for suffix := 2; used[candidate]; suffix++ {
			candidate = fmt.Sprintf("%s-%d", name, suffix)
		}
		used[candidate] = true
		proxy["name"] = candidate
	}
}

func inboundClashProxies(inbound *model.Inbound, requestHost string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0)
	settings := decodeObject(inbound.Settings)
	stream := decodeObject(inbound.StreamSettings)
	server := requestHost
	if inbound.Listen != "" && inbound.Listen != "0.0.0.0" && inbound.Listen != "::" && inbound.Listen != "[::]" {
		server = strings.Trim(inbound.Listen, "[]")
	}
	if server == "" {
		return nil
	}

	base := map[string]interface{}{"server": server, "port": inbound.Port, "udp": true}
	security := stringValue(stream["security"])
	network := stringValue(stream["network"])
	if network == "" {
		network = "tcp"
	}
	streamName := ""
	if inbound.Remark != "" {
		streamName = inbound.Remark
	} else {
		streamName = fmt.Sprintf("%s-%d", inbound.Protocol, inbound.Port)
	}

	switch inbound.Protocol {
	case model.VMess:
		users := objectArray(settings["clients"])
		for i, user := range users {
			proxy := cloneMap(base)
			proxy["name"] = numberedProxyName(streamName, i, len(users))
			proxy["type"] = "vmess"
			proxy["uuid"] = stringValue(user["id"])
			proxy["alterId"] = intValue(user["alterId"])
			proxy["cipher"] = "auto"
			addTransport(proxy, network, stream)
			addTLS(proxy, security, stream)
			if proxy["uuid"] != "" {
				results = append(results, proxy)
			}
		}
		return results
	case model.VLESS:
		users := objectArray(settings["clients"])
		for i, user := range users {
			proxy := cloneMap(base)
			proxy["name"] = numberedProxyName(streamName, i, len(users))
			proxy["type"] = "vless"
			proxy["uuid"] = stringValue(user["id"])
			if flow := stringValue(user["flow"]); flow != "" {
				proxy["flow"] = flow
			}
			addTransport(proxy, network, stream)
			addTLS(proxy, security, stream)
			if proxy["uuid"] != "" {
				results = append(results, proxy)
			}
		}
		return results
	case model.Trojan:
		users := objectArray(settings["clients"])
		for i, user := range users {
			proxy := cloneMap(base)
			proxy["name"] = numberedProxyName(streamName, i, len(users))
			proxy["type"] = "trojan"
			proxy["password"] = stringValue(user["password"])
			addTransport(proxy, network, stream)
			addTLS(proxy, security, stream)
			if proxy["password"] != "" {
				results = append(results, proxy)
			}
		}
		return results
	case model.Shadowsocks:
		proxy := cloneMap(base)
		proxy["name"] = streamName
		proxy["type"] = "ss"
		proxy["cipher"] = stringValue(settings["method"])
		proxy["password"] = stringValue(settings["password"])
		if proxy["cipher"] == "" || proxy["password"] == "" {
			return nil
		}
		return []map[string]interface{}{proxy}
	case model.Hysteria:
		users := objectArray(settings["users"])
		for i, user := range users {
			proxy := cloneMap(base)
			proxy["name"] = numberedProxyName(streamName, i, len(users))
			proxy["type"] = "hysteria2"
			proxy["password"] = stringValue(user["auth"])
			addTLS(proxy, security, stream)
			if proxy["password"] != "" {
				results = append(results, proxy)
			}
		}
		return results
	default:
		return nil
	}
}

func addTLS(proxy map[string]interface{}, security string, stream map[string]interface{}) {
	if security != "tls" && security != "reality" {
		return
	}
	proxy["tls"] = true
	if security == "reality" {
		reality := objectValue(stream["realitySettings"])
		proxy["client-fingerprint"] = defaultString(stringValue(reality["fingerprint"]), "chrome")
		serverNames := stringArray(reality["serverNames"])
		realityOpts := map[string]interface{}{}
		if len(serverNames) > 0 {
			proxy["servername"] = serverNames[0]
		}
		if key := stringValue(reality["publicKey"]); key != "" {
			realityOpts["public-key"] = key
		}
		shortIDs := stringArray(reality["shortIds"])
		if len(shortIDs) > 0 && shortIDs[0] != "" {
			realityOpts["short-id"] = shortIDs[0]
		}
		proxy["reality-opts"] = realityOpts
		return
	}
	tls := objectValue(stream["tlsSettings"])
	if name := stringValue(tls["serverName"]); name != "" {
		if proxy["type"] == "hysteria2" {
			proxy["sni"] = name
		} else {
			proxy["servername"] = name
		}
	}
	if alpn := stringArray(tls["alpn"]); len(alpn) > 0 {
		proxy["alpn"] = alpn
	}
}

func addTransport(proxy map[string]interface{}, network string, stream map[string]interface{}) {
	if network == "" || network == "tcp" {
		return
	}
	proxy["network"] = network
	settings := objectValue(stream[network+"Settings"])
	switch network {
	case "ws":
		ws := map[string]interface{}{}
		if path := stringValue(settings["path"]); path != "" {
			ws["path"] = path
		}
		headers := objectValue(settings["headers"])
		if len(headers) > 0 {
			ws["headers"] = headers
		}
		proxy["ws-opts"] = ws
	case "grpc":
		proxy["grpc-opts"] = map[string]interface{}{"grpc-service-name": stringValue(settings["serviceName"])}
	case "xhttp":
		opts := map[string]interface{}{}
		if path := stringValue(settings["path"]); path != "" {
			opts["path"] = path
		}
		if mode := stringValue(settings["mode"]); mode != "" {
			opts["mode"] = mode
		}
		if host := stringValue(settings["host"]); host != "" {
			opts["host"] = host
		}
		proxy["xhttp-opts"] = opts
	case "http":
		opts := map[string]interface{}{}
		if path := stringValue(settings["path"]); path != "" {
			opts["path"] = []string{path}
		}
		if hosts := stringArray(settings["host"]); len(hosts) > 0 {
			opts["headers"] = map[string]interface{}{"Host": hosts}
		}
		proxy["http-opts"] = opts
	}
}

func decodeObject(raw string) map[string]interface{} {
	result := map[string]interface{}{}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func objectValue(value interface{}) map[string]interface{} {
	if object, ok := value.(map[string]interface{}); ok {
		return object
	}
	return map[string]interface{}{}
}

func objectArray(value interface{}) []map[string]interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]interface{}); ok {
			result = append(result, object)
		}
	}
	return result
}

func stringArray(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && text != "" {
			result = append(result, text)
		}
	}
	return result
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func intValue(value interface{}) int {
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	case json.Number:
		parsed, _ := number.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func cloneMap(source map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{}, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func numberedProxyName(name string, index, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s-%d", name, index+1)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
