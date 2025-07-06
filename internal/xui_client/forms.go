package xui_client

import (
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type InboundAddForm struct {
	Remark         string
	Port           string
	Protocol       string
	Settings       string // JSON-строка
	StreamSettings string // JSON-строка
	Sniffing       string // JSON-строка
	Enable         string // "true" или "false"
}

func (f *InboundAddForm) ToFormData() string {
	data := url.Values{}
	data.Set("remark", f.Remark)
	data.Set("port", f.Port)
	data.Set("protocol", f.Protocol)
	data.Set("settings", f.Settings)
	data.Set("streamSettings", f.StreamSettings)
	data.Set("sniffing", f.Sniffing)
	data.Set("enable", f.Enable)
	return data.Encode()
}

// daysValid - срок действия в днях, если 0 или меньше, то бессрочно (expiryTime=0)
func GenerateRandomInboundForm(daysValid int) *InboundAddForm {
	rand.Seed(time.Now().UnixNano())
	randomPort := 20000 + rand.Intn(40000)
	randomId := uuid.New().String()
	randomEmail := uuid.New().String()[:8]
	randomSubId := uuid.New().String()[:16]
	expiryTime := 0 // по умолчанию бессрочно
	if daysValid > 0 {
		expiryTime = int(time.Now().Add(time.Duration(daysValid)*24*time.Hour).Unix() * 1000)
	}
	return &InboundAddForm{
		Remark:   "auto-generated",
		Port:     fmt.Sprintf("%d", randomPort),
		Protocol: "vless",
		Settings: fmt.Sprintf(`{"clients":[{"id":"%s","flow":"","email":"%s","limitIp":0,"totalGB":0,"expiryTime":%d,"enable":true,"tgId":"","subId":"%s","comment":"","reset":0}],"decryption":"none","fallbacks":[]}`,
			randomId, randomEmail, expiryTime, randomSubId),
		StreamSettings: `{"network":"tcp","security":"reality","externalProxy":[],"realitySettings":{"show":false,"xver":0,"dest":"yahoo.com:443","serverNames":["yahoo.com","www.yahoo.com"],"privateKey":"SBLy12PtgIQhVDQk9kgQm9W3ubthkpKOiZsiSepo6WE","minClient":"","maxClient":"","maxTimediff":0,"shortIds":["cb1387d77a62","c1f87d4c98","ce5356c7748f66","fc97e9","19","a192c5c8","c9d6","3e2bb7312ad2e425"],"settings":{"publicKey":"RV7AJOzhT5kEBckDgTN7jDbmRFZEdg8M5xXmDQJNB1Y","fingerprint":"chrome","serverName":"","spiderX":"/"}},"tcpSettings":{"acceptProxyProtocol":true,"header":{"type":"http","request":{"version":"1.1","method":"GET","path":["/"],"headers":{}},"response":{"version":"1.1","status":"200","reason":"OK","headers":{}}}}}`,
		Sniffing:       `{"enabled":false,"destOverride":["http","tls","quic","fakedns"],"metadataOnly":false,"routeOnly":false}`,
		Enable:         "true",
	}
}

// Генерация inbound без пользователей
func GenerateEmptyInboundForm(port int, remark string) *InboundAddForm {
	return &InboundAddForm{
		Remark:         remark,
		Port:           fmt.Sprintf("%d", port),
		Protocol:       "vless",
		Settings:       `{"clients":[],"decryption":"none","fallbacks":[]}`,
		StreamSettings: `{"network":"tcp","security":"reality","externalProxy":[],"realitySettings":{"show":false,"xver":0,"dest":"yahoo.com:443","serverNames":["yahoo.com","www.yahoo.com"],"privateKey":"SBLy12PtgIQhVDQk9kgQm9W3ubthkpKOiZsiSepo6WE","minClient":"","maxClient":"","maxTimediff":0,"shortIds":["cb1387d77a62","c1f87d4c98","ce5356c7748f66","fc97e9","19","a192c5c8","c9d6","3e2bb7312ad2e425"],"settings":{"publicKey":"RV7AJOzhT5kEBckDgTN7jDbmRFZEdg8M5xXmDQJNB1Y","fingerprint":"chrome","serverName":"","spiderX":"/"}},"tcpSettings":{"acceptProxyProtocol":true,"header":{"type":"http","request":{"version":"1.1","method":"GET","path":["/"],"headers":{}},"response":{"version":"1.1","status":"200","reason":"OK","headers":{}}}}}`,
		Sniffing:       `{"enabled":false,"destOverride":["http","tls","quic","fakedns"],"metadataOnly":false,"routeOnly":false}`,
		Enable:         "true",
	}
}

type AddClientForm struct {
	Id       int
	Settings string // JSON-строка с clients
}

func (f *AddClientForm) ToFormData() string {
	data := url.Values{}
	data.Set("id", fmt.Sprintf("%d", f.Id))
	data.Set("settings", f.Settings)
	return data.Encode()
}

// Генерация JSON settings для одного пользователя
func GenerateClientSettings(id, email, subId string, totalGB int64, expiryTime int64, tgId int64) string {
	return fmt.Sprintf(`{"clients": [{"id": "%s", "flow": "", "email": "%s", "limitIp": 0, "totalGB": %d, "expiryTime": %d, "enable": true, "tgId": %d, "subId": "%s", "comment": "", "reset": 0}]}`,
		id, email, totalGB, expiryTime, tgId, subId)
}

// Генерация JSON settings для одного случайного пользователя
func GenerateRandomClientSettings(expiryDays int) (clientId, email, subId string, settings string) {
	rand.Seed(time.Now().UnixNano())
	clientId = uuid.New().String()
	email = uuid.New().String()[:8]
	subId = uuid.New().String()[:16]
	tgId := rand.Int63n(100000) + 10000
	totalGB := int64(10 * 1024 * 1024 * 1024) // 10 ГБ
	expiryTime := int64(0)
	if expiryDays > 0 {
		expiryTime = time.Now().Add(time.Duration(expiryDays)*24*time.Hour).Unix() * 1000
	}
	settings = fmt.Sprintf(`{"clients": [{"id": "%s", "flow": "", "email": "%s", "limitIp": 0, "totalGB": %d, "expiryTime": %d, "enable": true, "tgId": %d, "subId": "%s", "comment": "", "reset": 0}]}`,
		clientId, email, totalGB, expiryTime, tgId, subId)
	return
}
