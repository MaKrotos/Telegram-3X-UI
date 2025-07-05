package services

import (
	"TelegramXUI/internal/contracts"
	"TelegramXUI/internal/xui_client"
	"fmt"
	"log"
	"sync"
	"time"
)

type HostMonitorService struct {
	serverService  *XUIServerService
	adminService   *AdminService
	telegramClient contracts.TelegramMessageSender
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	isRunning      bool
	mu             sync.RWMutex
}

type HostStatus struct {
	ServerID   int
	ServerName string
	ServerURL  string
	IsActive   bool
	Error      string
	CheckedAt  time.Time
}

func NewHostMonitorService(
	serverService *XUIServerService,
	adminService *AdminService,
	telegramClient contracts.TelegramMessageSender,
	checkInterval time.Duration,
) *HostMonitorService {
	return &HostMonitorService{
		serverService:  serverService,
		adminService:   adminService,
		telegramClient: telegramClient,
		checkInterval:  checkInterval,
		stopChan:       make(chan struct{}),
	}
}

// Start запускает мониторинг хостов
func (s *HostMonitorService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("мониторинг уже запущен")
	}

	// Создаем новый канал для остановки
	s.stopChan = make(chan struct{})
	s.isRunning = true
	s.wg.Add(1)

	go s.monitorLoop()

	log.Printf("[HostMonitor] Мониторинг хостов запущен с интервалом %v", s.checkInterval)
	return nil
}

// Stop останавливает мониторинг хостов
func (s *HostMonitorService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("мониторинг не запущен")
	}

	// Проверяем, не закрыт ли уже канал
	select {
	case <-s.stopChan:
		// Канал уже закрыт
		log.Printf("[HostMonitor] Канал остановки уже закрыт")
	default:
		// Закрываем канал
		close(s.stopChan)
	}

	s.wg.Wait()
	s.isRunning = false

	log.Printf("[HostMonitor] Мониторинг хостов остановлен")
	return nil
}

// IsRunning проверяет, запущен ли мониторинг
func (s *HostMonitorService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// monitorLoop основной цикл мониторинга
func (s *HostMonitorService) monitorLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	// Выполняем первую проверку сразу
	s.checkAllHosts()

	for {
		select {
		case <-ticker.C:
			s.checkAllHosts()
		case <-s.stopChan:
			return
		}
	}
}

// checkAllHosts проверяет все активные хосты
func (s *HostMonitorService) checkAllHosts() {
	log.Printf("[HostMonitor] Начинаем проверку всех активных хостов")

	servers, err := s.serverService.GetActiveServers()
	if err != nil {
		log.Printf("[HostMonitor] Ошибка получения активных серверов: %v", err)
		return
	}

	if len(servers) == 0 {
		log.Printf("[HostMonitor] Нет активных серверов для проверки")
		return
	}

	var wg sync.WaitGroup
	statusChan := make(chan HostStatus, len(servers))

	// Запускаем проверку каждого хоста в отдельной горутине
	for _, server := range servers {
		wg.Add(1)
		go func(srv *XUIServer) {
			defer wg.Done()
			status := s.checkHost(srv)
			statusChan <- status
		}(server)
	}

	// Ждем завершения всех проверок
	go func() {
		wg.Wait()
		close(statusChan)
	}()

	// Обрабатываем результаты
	var inactiveHosts []HostStatus
	for status := range statusChan {
		if !status.IsActive {
			inactiveHosts = append(inactiveHosts, status)
		}
	}

	// Отправляем уведомления о неактивных хостах
	if len(inactiveHosts) > 0 {
		s.notifyAdminsAboutInactiveHosts(inactiveHosts)
	}

	log.Printf("[HostMonitor] Проверка завершена. Найдено неактивных хостов: %d", len(inactiveHosts))
}

// checkHost проверяет конкретный хост
func (s *HostMonitorService) checkHost(server *XUIServer) HostStatus {
	status := HostStatus{
		ServerID:   server.ID,
		ServerName: server.ServerName,
		ServerURL:  server.ServerURL,
		IsActive:   true,
		CheckedAt:  time.Now(),
	}

	log.Printf("[HostMonitor] Проверяем хост: %s (%s)", server.ServerName, server.ServerURL)

	// Создаем клиент для проверки
	client := xui_client.NewClient(server.ServerURL, server.Username, server.Password)

	// Пытаемся авторизоваться
	err := client.Login()
	if err != nil {
		status.IsActive = false
		status.Error = fmt.Sprintf("Ошибка авторизации: %v", err)
		log.Printf("[HostMonitor] Хост %s неактивен: %v", server.ServerName, err)

		// Обновляем статус в базе данных
		if updateErr := s.serverService.SetServerStatus(server.ID, false); updateErr != nil {
			log.Printf("[HostMonitor] Ошибка обновления статуса хоста %d: %v", server.ID, updateErr)
		}

		return status
	}

	// Пытаемся получить список пользователей для проверки API
	_, err = client.GetUsers()
	if err != nil {
		status.IsActive = false
		status.Error = fmt.Sprintf("Ошибка API: %v", err)
		log.Printf("[HostMonitor] Хост %s неактивен (ошибка API): %v", server.ServerName, err)

		// Обновляем статус в базе данных
		if updateErr := s.serverService.SetServerStatus(server.ID, false); updateErr != nil {
			log.Printf("[HostMonitor] Ошибка обновления статуса хоста %d: %v", server.ID, updateErr)
		}

		return status
	}

	log.Printf("[HostMonitor] Хост %s активен", server.ServerName)
	return status
}

// notifyAdminsAboutInactiveHosts уведомляет администраторов о неактивных хостах
func (s *HostMonitorService) notifyAdminsAboutInactiveHosts(inactiveHosts []HostStatus) {
	// Получаем информацию о глобальном администраторе
	adminInfo := s.adminService.GetGlobalAdminInfo()
	tgID, ok := adminInfo["tg_id"].(int64)
	if !ok || tgID == 0 {
		log.Printf("[HostMonitor] Не удалось получить Telegram ID администратора")
		return
	}

	// Формируем сообщение
	message := "🚨 **ВНИМАНИЕ! Обнаружены неактивные хосты:**\n\n"

	for _, host := range inactiveHosts {
		message += fmt.Sprintf("❌ **%s** (`%s`)\n", host.ServerName, host.ServerURL)
		message += fmt.Sprintf("   Ошибка: %s\n", host.Error)
		message += fmt.Sprintf("   Проверено: %s\n\n", host.CheckedAt.Format("02.01.2006 15:04:05"))
	}

	message += "Хосты автоматически отключены и не будут использоваться для создания VPN."

	// Отправляем уведомление администратору
	if err := s.telegramClient.SendMessage(tgID, message); err != nil {
		log.Printf("[HostMonitor] Ошибка отправки уведомления администратору: %v", err)
	} else {
		log.Printf("[HostMonitor] Уведомление о неактивных хостах отправлено администратору %d", tgID)
	}
}

// CheckHostNow выполняет немедленную проверку конкретного хоста
func (s *HostMonitorService) CheckHostNow(serverID int) (*HostStatus, error) {
	server, err := s.serverService.GetServerByID(serverID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения сервера: %w", err)
	}

	if server == nil {
		return nil, fmt.Errorf("сервер с ID %d не найден", serverID)
	}

	status := s.checkHost(server)
	return &status, nil
}

// GetMonitoringStatus возвращает статус мониторинга
func (s *HostMonitorService) GetMonitoringStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"is_running":     s.isRunning,
		"check_interval": s.checkInterval.String(),
	}
}
