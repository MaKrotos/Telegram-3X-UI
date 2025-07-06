package contracts

import "time"

// --- UserStateService ---
type UserStateInfo struct {
	ID                     int
	TelegramID             int64
	Username               string
	FirstName              string
	LastName               string
	State                  string
	ExpectedAction         string
	StateChangedAt         time.Time
	StateReason            string
	StateChangedByTgID     int64
	StateChangedByUsername string
	StateExpiresAt         *time.Time
	StateMetadata          map[string]interface{}
	CreatedAt              time.Time
	UpdatedAt              time.Time
	LastActivity           time.Time
}

type UserStateService interface {
	GetUserState(telegramID int64) (*UserStateInfo, error)
	CanUserPerformAction(telegramID int64) (bool, string, error)
}

// --- AdminService ---
type AdminService interface {
	IsGlobalAdmin(tgID int64) bool
}

// --- XUIHostAddService ---
type XUIHostData struct {
	Host      string
	Login     string
	Password  string
	SecretKey string
}

type XUIHostAddService interface {
	StartAddHostProcess(telegramID int64, username string) error
	ProcessHostData(telegramID int64, message string, username string) (*XUIHostData, error)
	CancelAddHostProcess(telegramID int64, username string) error
	GetAddHostInstructions() string
	IsInAddHostState(telegramID int64) (bool, error)
}

// --- ExtensibleStateService ---
type ExtensibleStateService interface {
	// Добавьте методы, которые реально используются в message_handler.go
}
