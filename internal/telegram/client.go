package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"TelegramXUI/internal/contracts"
)

// TelegramClient представляет клиент для работы с Telegram Bot API
type TelegramClient struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
}

// Message представляет сообщение Telegram
type Message struct {
	MessageID         int                `json:"message_id"`
	From              contracts.User     `json:"from"`
	Chat              Chat               `json:"chat"`
	Date              int                `json:"date"`
	Text              string             `json:"text"`
	SuccessfulPayment *SuccessfulPayment `json:"successful_payment,omitempty"`
}

// User представляет пользователя Telegram
type User = contracts.User

// Chat представляет чат Telegram
type Chat struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// CallbackQuery представляет callback query от inline кнопки
type CallbackQuery struct {
	ID   string `json:"id"`
	From User   `json:"from"`
	Data string `json:"data"`
}

// PreCheckoutQuery представляет pre_checkout_query для поддержки Telegram Payments/Stars
type PreCheckoutQuery struct {
	ID             string         `json:"id"`
	From           contracts.User `json:"from"`
	Currency       string         `json:"currency"`
	TotalAmount    int            `json:"total_amount"`
	InvoicePayload string         `json:"invoice_payload"`
}

// Update представляет обновление от Telegram
type Update struct {
	UpdateID         int               `json:"update_id"`
	Message          *Message          `json:"message,omitempty"`
	CallbackQuery    *CallbackQuery    `json:"callback_query,omitempty"`
	PreCheckoutQuery *PreCheckoutQuery `json:"pre_checkout_query,omitempty"`
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

// SuccessfulPayment представляет информацию о успешной оплате через Telegram Payments/Stars
type SuccessfulPayment struct {
	Currency                string `json:"currency"`
	TotalAmount             int    `json:"total_amount"`
	InvoicePayload          string `json:"invoice_payload"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
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
		"parse_mode":   "HTML",
		"reply_markup": keyboard,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	log.Printf("[TelegramAPI] Отправка сообщения с клавиатурой: chat_id=%d, text=\"%s\"", chatID, text)
	log.Printf("[TelegramAPI] JSON запрос: %s", string(jsonData))

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

// AnswerCallbackQuery отвечает на callback query
func (c *TelegramClient) AnswerCallbackQuery(callbackQueryID, text string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("callback_query_id", callbackQueryID)
	if text != "" {
		params.Set("text", text)
	}

	resp, err := c.HTTPClient.PostForm(c.BaseURL+"/answerCallbackQuery", params)
	if err != nil {
		return nil, fmt.Errorf("ошибка ответа на callback query: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return result, nil
}

// SendInvoice отправляет инвойс пользователю через Telegram Payments/Stars
func (c *TelegramClient) SendInvoice(chatID int, title, description, payload, providerToken, currency string, prices []LabeledPrice, isTest bool) error {
	invoice := map[string]interface{}{
		"chat_id":     chatID,
		"title":       title,
		"description": description,
		"payload":     payload,
		"currency":    currency,
		"prices":      prices,
	}
	if currency != "XTR" {
		invoice["provider_token"] = providerToken
		if isTest {
			invoice["provider_token"] = "TEST:TOKEN"
		}
	}
	jsonData, err := json.Marshal(invoice)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга инвойса: %w", err)
	}
	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/sendInvoice",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("ошибка отправки инвойса: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}
	log.Printf("[TelegramAPI] Ответ на sendInvoice: %s", string(bodyBytes))
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return fmt.Errorf("ошибка Telegram API: %v", result["description"])
	}
	return nil
}

// LabeledPrice описывает цену для инвойса
// https://core.telegram.org/bots/api#labeledprice
type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

// AnswerPreCheckoutQuery отвечает на pre_checkout_query
func (c *TelegramClient) AnswerPreCheckoutQuery(preCheckoutQueryID string, ok bool, errorMessage string) error {
	data := map[string]interface{}{
		"pre_checkout_query_id": preCheckoutQueryID,
		"ok":                    ok,
	}
	if !ok && errorMessage != "" {
		data["error_message"] = errorMessage
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}
	log.Printf("[DEBUG] Отправка answerPreCheckoutQuery: %s", string(jsonData))
	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/answerPreCheckoutQuery",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("[ERROR] Ошибка отправки запроса answerPreCheckoutQuery: %v", err)
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Ошибка чтения ответа answerPreCheckoutQuery: %v", err)
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}
	log.Printf("[TelegramAPI] Ответ на answerPreCheckoutQuery: %s", string(bodyBytes))
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		log.Printf("[ERROR] Ошибка декодирования ответа answerPreCheckoutQuery: %v", err)
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}
	if ok, exists := result["ok"].(bool); !exists || !ok {
		log.Printf("[ERROR] Ошибка Telegram API в answerPreCheckoutQuery: %v", result["description"])
		return fmt.Errorf("ошибка Telegram API: %v", result["description"])
	}
	return nil
}

// RefundStarPayment осуществляет возврат звёзд пользователю по payment_charge_id
func (c *TelegramClient) RefundStarPayment(userID int64, telegramPaymentChargeID string, amount int, reason string) error {
	data := map[string]interface{}{
		"user_id":                    userID,
		"telegram_payment_charge_id": telegramPaymentChargeID,
	}
	if amount > 0 {
		data["amount"] = amount
	}
	if reason != "" {
		data["reason"] = reason
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}
	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/refundStarPayment",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}
	log.Printf("[TelegramAPI] Ответ на refundStarPayment: %s", string(bodyBytes))
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}
	if ok, exists := result["ok"].(bool); !exists || !ok {
		return fmt.Errorf("ошибка Telegram API: %v", result["description"])
	}
	return nil
}

// SetMyCommands устанавливает список команд бота
func (c *TelegramClient) SetMyCommands(commands []map[string]string) error {
	data := map[string]interface{}{"commands": commands}
	jsonData, _ := json.Marshal(data)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/setMyCommands", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SetMyDescription устанавливает описание бота
func (c *TelegramClient) SetMyDescription(description string) error {
	data := map[string]string{"description": description}
	jsonData, _ := json.Marshal(data)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/setMyDescription", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SetMyShortDescription устанавливает короткое описание бота
func (c *TelegramClient) SetMyShortDescription(shortDescription string) error {
	data := map[string]string{"short_description": shortDescription}
	jsonData, _ := json.Marshal(data)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/setMyShortDescription", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SetMyAboutText устанавливает текст about (в окне профиля)
func (c *TelegramClient) SetMyAboutText(about string) error {
	data := map[string]string{"about": about}
	jsonData, _ := json.Marshal(data)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/setMyAboutText", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SetMyProfilePhoto устанавливает фото профиля бота (путь к файлу)
func (c *TelegramClient) SetMyProfilePhoto(photoPath string) error {
	file, err := os.Open(photoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := io.MultiWriter(body)
	io.Copy(writer, file)

	req, err := http.NewRequest("POST", c.BaseURL+"/setMyProfilePhoto", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
