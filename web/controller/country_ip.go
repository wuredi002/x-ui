package controller

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const countryLookupUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.87 Safari/537.36"

const nodeNameAuthor = "wuredi002"
const nodeNameNetworkTag = "网路跳越"

var countryIPCache struct {
	sync.Mutex
	country string
	ip      string
	expires time.Time
}

var publicIPv4Services = []string{
	"https://test.ipw.cn",
	"https://icanhazip.com",
	"https://api.ipify.org",
	"https://ident.me",
	"https://ipecho.net/plain",
	"https://checkip.amazonaws.com",
	"https://bot.whatismyipaddress.com",
	"https://ipinfo.io/ip",
	"https://myexternalip.com/raw",
	"https://ifconfig.me/ip",
	"https://wgetip.com",
	"https://ip.seeip.org",
}

func countryIPPrefix(country, ip string) string {
	parts := make([]string, 0, 2)
	if country != "" {
		parts = append(parts, country)
	}
	if ip != "" {
		parts = append(parts, ip)
	}
	parts = append(parts, nodeNameAuthor, nodeNameNetworkTag)
	return strings.Join(parts, "-")
}

func detectCountryAndPublicIPv4() (string, string) {
	countryIPCache.Lock()
	defer countryIPCache.Unlock()
	if time.Now().Before(countryIPCache.expires) {
		return countryIPCache.country, countryIPCache.ip
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	publicIP := ""
	for _, endpoint := range publicIPv4Services {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 128))
		response.Body.Close()
		if readErr != nil || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			continue
		}
		candidate := strings.TrimSpace(string(body))
		parsed := net.ParseIP(candidate)
		if parsed != nil && parsed.To4() != nil {
			publicIP = parsed.To4().String()
			break
		}
		if ctx.Err() != nil {
			break
		}
	}

	country := ""
	if publicIP != "" && ctx.Err() == nil {
		country = lookupIPCountry(ctx, client, publicIP)
	}
	countryIPCache.country = country
	countryIPCache.ip = publicIP
	countryIPCache.expires = time.Now().Add(30 * time.Minute)
	return country, publicIP
}

func lookupIPCountry(ctx context.Context, client *http.Client, ip string) string {
	endpoint := "http://ip-api.com/json/" + url.PathEscape(ip) + "?lang=zh-CN&fields=status,country"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	request.Header.Set("User-Agent", countryLookupUserAgent)
	response, err := client.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ""
	}
	var result struct {
		Status  string `json:"status"`
		Country string `json:"country"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result); err != nil || result.Status != "success" {
		return ""
	}
	return strings.TrimSpace(result.Country)
}
