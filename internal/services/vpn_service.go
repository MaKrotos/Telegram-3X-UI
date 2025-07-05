package services

import (
	"fmt"
	"log"
	"math/rand"

	"TelegramXUI/internal/config"
	"TelegramXUI/internal/xui_client"
)

// VPNService предоставляет методы для работы с VPN
type VPNService struct {
	xuiClient *xui_client.Client
}

// NewVPNService создает новый сервис VPN
func NewVPNService(xuiClient *xui_client.Client) *VPNService {
	return &VPNService{
		xuiClient: xuiClient,
	}
}

// CheckStatus проверяет статус подключения к x-ui
func (s *VPNService) CheckStatus() error {
	if s.xuiClient == nil {
		return fmt.Errorf("x-ui клиент не инициализирован")
	}
	return s.xuiClient.Login()
}

// CreateVPNForUser создает VPN подключение для пользователя
func (s *VPNService) CreateVPNForUser(telegramUserID int64, userName string, vpnConfig *config.VPNConfig) (map[string]interface{}, error) {
	if s.xuiClient == nil {
		return nil, fmt.Errorf("x-ui клиент не инициализирован")
	}

	// Логируем в x-ui
	if err := s.xuiClient.Login(); err != nil {
		return nil, fmt.Errorf("ошибка входа в x-ui: %w", err)
	}

	// Используем случайный порт для inbound из конфигурации
	portRange := vpnConfig.PortRangeEnd - vpnConfig.PortRangeStart
	port := vpnConfig.PortRangeStart + rand.Intn(portRange)
	inboundName := fmt.Sprintf("VPN для %s (ID: %d)", userName, telegramUserID)

	log.Printf("[VPN] Создание inbound для пользователя %s (ID: %d) на порту %d", userName, telegramUserID, port)

	emptyInbound := xui_client.GenerateEmptyInboundForm(port, inboundName)
	inboundId, err := s.xuiClient.AddInbound(emptyInbound)
	if err != nil || inboundId == 0 {
		return nil, fmt.Errorf("ошибка создания inbound: %w", err)
	}

	log.Printf("[VPN] Inbound создан успешно: id=%d, port=%d", inboundId, port)

	// Создаем случайного клиента
	clientId, email, subId, settings := xui_client.GenerateRandomClientSettings(10)
	addClientForm := &xui_client.AddClientForm{
		Id:       inboundId,
		Settings: settings,
	}

	if err := s.xuiClient.AddClientToInbound(addClientForm); err != nil {
		return nil, fmt.Errorf("ошибка добавления клиента: %w", err)
	}

	log.Printf("[VPN] Клиент добавлен успешно: id=%d, email=%s, subId=%s", clientId, email, subId)

	// Формируем данные для подключения
	connectionData := map[string]interface{}{
		"inbound_id":  inboundId,
		"client_id":   clientId,
		"email":       email,
		"sub_id":      subId,
		"port":        port,
		"settings":    settings,
		"user_name":   userName,
		"telegram_id": telegramUserID,
		"server_ip":   vpnConfig.ServerIP,
	}

	return connectionData, nil
}
