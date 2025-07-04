package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// TelegramClient представляет клиент для работы с Telegram Bot API
type TelegramClient struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
}

// Message представляет сообщение Telegram
type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Date      int    `json:"date"`
	Text      string `json:"text"`
}

// User представляет пользователя Telegram
type User struct {
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// Chat представляет чат Telegram
type Chat struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// Update представляет обновление от Telegram
type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
}

// GetUpdatesResponse представляет ответ на запрос обновлений
type GetUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// SendMessageResponse представляет ответ на отправку сообщения
type SendMessageResponse struct {
	OK     bool    `json:"ok"`
	Result Message `json:"result"`
}

// SendMessageRequest представляет запрос на отправку сообщения
type SendMessageRequest struct {
	ChatID      int    `json:"chat_id"`
	Text        string `json:"text"`
	ParseMode   string `json:"parse_mode,omitempty"`
	ReplyMarkup string `json:"reply_markup,omitempty"`
}

// InlineKeyboardButton представляет кнопку inline клавиатуры
type InlineKeyboardButton struct {
	Text         string  `json:"text"`
	URL          string  `json:"url,omitempty"`
	CallbackData string  `json:"callback_data,omitempty"`
	WebApp       *WebApp `json:"web_app,omitempty"`
}

// WebApp представляет WebApp для кнопки
type WebApp struct {
	URL string `json:"url"`
}

// InlineKeyboardMarkup представляет inline клавиатуру
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// NewClient создает новый экземпляр TelegramClient
func NewClient(token string) *TelegramClient {
	return &TelegramClient{
		Token:   token,
		BaseURL: "https://api.telegram.org/bot" + token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMe получает информацию о боте
func (c *TelegramClient) GetMe() (map[string]interface{}, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/getMe")
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса getMe: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// GetUpdates получает обновления от Telegram
func (c *TelegramClient) GetUpdates(offset, limit int) (*GetUpdatesResponse, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	url := c.BaseURL + "/getUpdates"
	if len(params) > 0 {
		url += "?" + params.Encode()
	}

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса getUpdates: %w", err)
	}
	defer resp.Body.Close()

	var result GetUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return &result, nil
}

// SendMessage отправляет сообщение в чат
func (c *TelegramClient) SendMessage(chatID int, text string, parseMode string) (*SendMessageResponse, error) {
	request := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	log.Printf("[TelegramAPI] Отправка сообщения: chat_id=%d, text=\"%s\", parse_mode=\"%s\"", chatID, text, parseMode)

	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/sendMessage",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки сообщения: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа для детального логирования
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	log.Printf("[TelegramAPI] Ответ от Telegram API: %s", string(bodyBytes))

	var result SendMessageResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if result.OK {
		log.Printf("[TelegramAPI] Сообщение отправлено успешно: message_id=%d", result.Result.MessageID)
	} else {
		// Пытаемся извлечь детали ошибки
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil {
			if description, ok := errorResponse["description"].(string); ok {
				log.Printf("[TelegramAPI] Ошибка отправки сообщения: %s", description)
				return &result, fmt.Errorf("Telegram API error: %s", description)
			}
		}
		log.Printf("[TelegramAPI] Ошибка отправки сообщения: %v", result)
		return &result, fmt.Errorf("Telegram API вернул ошибку: OK=false")
	}

	return &result, nil
}

// SendMessageHTML отправляет HTML-сообщение
func (c *TelegramClient) SendMessageHTML(chatID int, text string) (*SendMessageResponse, error) {
	log.Printf("[TelegramAPI] Отправка HTML сообщения: chat_id=%d", chatID)
	return c.SendMessage(chatID, text, "HTML")
}

// SendMessageMarkdown отправляет Markdown-сообщение
func (c *TelegramClient) SendMessageMarkdown(chatID int, text string) (*SendMessageResponse, error) {
	return c.SendMessage(chatID, text, "MarkdownV2")
}

// SendPhoto отправляет фото
func (c *TelegramClient) SendPhoto(chatID int, photoURL, caption string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("photo", photoURL)
	if caption != "" {
		params.Set("caption", caption)
	}

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/sendPhoto", params)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки фото: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// SendDocument отправляет документ
func (c *TelegramClient) SendDocument(chatID int, documentURL, caption string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("document", documentURL)
	if caption != "" {
		params.Set("caption", caption)
	}

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/sendDocument", params)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки документа: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// DeleteMessage удаляет сообщение
func (c *TelegramClient) DeleteMessage(chatID, messageID int) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("message_id", strconv.Itoa(messageID))

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/deleteMessage", params)
	if err != nil {
		return nil, fmt.Errorf("ошибка удаления сообщения: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// SetWebhook устанавливает webhook для бота
func (c *TelegramClient) SetWebhook(webhookURL string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("url", webhookURL)

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/setWebhook", params)
	if err != nil {
		return nil, fmt.Errorf("ошибка установки webhook: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// DeleteWebhook удаляет webhook
func (c *TelegramClient) DeleteWebhook() (map[string]interface{}, error) {
	resp, err := c.HTTPClient.Post(c.BaseURL+"/deleteWebhook", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка удаления webhook: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// GetWebhookInfo получает информацию о webhook
func (c *TelegramClient) GetWebhookInfo() (map[string]interface{}, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/getWebhookInfo")
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о webhook: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// ParseUpdate парсит обновление из JSON
func (c *TelegramClient) ParseUpdate(body io.Reader) (*Update, error) {
	var update Update
	if err := json.NewDecoder(body).Decode(&update); err != nil {
		return nil, fmt.Errorf("ошибка парсинга обновления: %w", err)
	}
	return &update, nil
}

// SendMessageWithKeyboard отправляет сообщение с inline клавиатурой
func (c *TelegramClient) SendMessageWithKeyboard(chatID int, text string, keyboard *InlineKeyboardMarkup) (*SendMessageResponse, error) {
	request := map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"reply_markup": keyboard,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	log.Printf("[TelegramAPI] Отправка сообщения с клавиатурой: chat_id=%d, text=\"%s\"", chatID, text)

	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/sendMessage",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки сообщения: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа для детального логирования
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	log.Printf("[TelegramAPI] Ответ от Telegram API: %s", string(bodyBytes))

	var result SendMessageResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if result.OK {
		log.Printf("[TelegramAPI] Сообщение с клавиатурой отправлено успешно: message_id=%d", result.Result.MessageID)
	} else {
		// Пытаемся извлечь детали ошибки
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil {
			if description, ok := errorResponse["description"].(string); ok {
				log.Printf("[TelegramAPI] Ошибка отправки сообщения с клавиатурой: %s", description)
				return &result, fmt.Errorf("Telegram API error: %s", description)
			}
		}
		log.Printf("[TelegramAPI] Ошибка отправки сообщения с клавиатурой: %v", result)
		return &result, fmt.Errorf("Telegram API вернул ошибку: OK=false")
	}

	return &result, nil
}

// SendMessageWithWebAppButton отправляет сообщение с WebApp кнопкой
func (c *TelegramClient) SendMessageWithWebAppButton(chatID int, text, buttonText, webAppURL string) (*SendMessageResponse, error) {
	log.Printf("[TelegramAPI] Отправка WebApp кнопки: chat_id=%d, button_text=\"%s\", webapp_url=\"%s\"", chatID, buttonText, webAppURL)

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text: buttonText,
					WebApp: &WebApp{
						URL: webAppURL,
					},
				},
			},
		},
	}

	return c.SendMessageWithKeyboard(chatID, text, keyboard)
}

// SendWelcomeMessageWithWebApp отправляет приветственное сообщение с WebApp кнопкой
func (c *TelegramClient) SendWelcomeMessageWithWebApp(chatID int, userName, webAppURL string) (*SendMessageResponse, error) {
	log.Printf("[TelegramAPI] Отправка приветственного сообщения с WebApp: chat_id=%d, user_name=\"%s\", webapp_url=\"%s\"", chatID, userName, webAppURL)

	message := fmt.Sprintf(`Привет, %s! 👋

Добро пожаловать в VPN Manager Bot!

Здесь вы можете:
• Управлять своими VPN подключениями
• Просматривать статистику
• Настраивать параметры

Нажмите кнопку ниже, чтобы открыть панель управления:`, userName)

	return c.SendMessageWithWebAppButton(chatID, message, "🚀 Открыть панель управления", webAppURL)
}
