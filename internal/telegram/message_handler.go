package telegram

import (
	"TelegramXUI/internal/config"
	"TelegramXUI/internal/contracts"
	"TelegramXUI/internal/services"
)

// MessageProcessor обрабатывает сообщения Telegram бота
// (реализация вынесена в отдельные файлы: command_handlers.go, payment_handlers.go, vpn_handlers.go, admin_handlers.go, utils.go)
type MessageProcessor struct {
	userStateService       contracts.UserStateService
	extensibleStateService contracts.ExtensibleStateService
	xuiHostAddService      contracts.XUIHostAddService
	adminService           contracts.AdminService
	xuiServerService       *services.XUIServerService
	hostMonitorService     *services.HostMonitorService
	userService            *UserService
	vpnConnectionService   *services.VPNConnectionService
	config                 *config.Config
	transactionService     *services.TransactionService
}

func NewMessageProcessor(
	userStateService contracts.UserStateService,
	extensibleStateService contracts.ExtensibleStateService,
	xuiHostAddService contracts.XUIHostAddService,
	adminService contracts.AdminService,
	xuiServerService *services.XUIServerService,
	hostMonitorService *services.HostMonitorService,
	userService *UserService,
	vpnConnectionService *services.VPNConnectionService,
	config *config.Config,
	transactionService *services.TransactionService,
) *MessageProcessor {
	return &MessageProcessor{
		userStateService:       userStateService,
		extensibleStateService: extensibleStateService,
		xuiHostAddService:      xuiHostAddService,
		adminService:           adminService,
		xuiServerService:       xuiServerService,
		hostMonitorService:     hostMonitorService,
		userService:            userService,
		vpnConnectionService:   vpnConnectionService,
		config:                 config,
		transactionService:     transactionService,
	}
}

// ProcessMessage теперь делегирует обработку в соответствующие модули
func (p *MessageProcessor) ProcessMessage(client *TelegramClient, update Update) error {
	// Реализация вынесена в отдельные файлы (command_handlers.go, payment_handlers.go и др.)
	return p.routeMessage(client, update)
}
