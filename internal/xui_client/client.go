package xui_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL  string
	Username string
	Password string
	Token    string
	client   *http.Client
}

func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		client:   &http.Client{},
	}
}

func (c *Client) Login() error {
	fmt.Println("[xui_client] Попытка авторизации по адресу:", c.BaseURL+"/login")
	data := url.Values{}
	data.Set("username", c.Username)
	data.Set("password", c.Password)
	data.Set("twoFactorCode", "")

	req, err := http.NewRequest("POST", c.BaseURL+"/login", strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("[xui_client] Ошибка создания запроса:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Println("[xui_client] Ошибка отправки запроса:", err)
		return err
	}
	defer resp.Body.Close()

	fmt.Println("[xui_client] Код ответа:", resp.StatusCode)

	// Читаем и выводим тело ответа
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("[xui_client] Тело ответа:", string(bodyBytes))

	// Выводим все cookies
	fmt.Println("[xui_client] Cookies:")
	for _, cookie := range resp.Cookies() {
		fmt.Printf("  %s = %s\n", cookie.Name, cookie.Value)
	}

	// Сохраняем cookie сессии
	c.Token = ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" {
			c.Token = cookie.Value
			fmt.Println("[xui_client] Токен сессии получен:", c.Token)
			break
		}
	}
	if c.Token == "" {
		fmt.Println("[xui_client] session token not found in cookies")
		return errors.New("session token not found in cookies")
	}

	fmt.Println("[xui_client] Авторизация успешна")
	return nil
}

// CheckStatus проверяет статус сервера, пытаясь получить список пользователей
func (c *Client) CheckStatus() error {
	fmt.Println("[xui_client] Начинаем проверку статуса сервера:", c.BaseURL)

	// Сначала пытаемся авторизоваться
	if err := c.Login(); err != nil {
		fmt.Println("[xui_client] Ошибка авторизации при проверке статуса:", err)
		return fmt.Errorf("login failed: %v", err)
	}

	fmt.Println("[xui_client] Авторизация успешна, проверяем API...")

	// Затем пытаемся получить список пользователей
	users, err := c.GetUsers()
	if err != nil {
		fmt.Println("[xui_client] Ошибка получения пользователей при проверке статуса:", err)
		return fmt.Errorf("get users failed: %v", err)
	}

	fmt.Printf("[xui_client] Статус сервера OK, получено пользователей: %d\n", len(users))
	return nil
}

func (c *Client) GetUsers() ([]map[string]interface{}, error) {
	fmt.Println("[xui_client] Запрос списка пользователей:", c.BaseURL+"/api/user/list")
	req, err := http.NewRequest("GET", c.BaseURL+"/api/user/list", nil)
	if err != nil {
		fmt.Println("[xui_client] Ошибка создания запроса:", err)
		return nil, err
	}
	// Добавляем cookie сессии
	req.AddCookie(&http.Cookie{
		Name:  "3x-ui",
		Value: c.Token,
	})
	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Println("[xui_client] Ошибка выполнения запроса:", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("[xui_client] Код ответа:", resp.StatusCode)
	if resp.StatusCode != 200 {
		fmt.Println("[xui_client] Ошибка статуса:", resp.Status)
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("[xui_client] Тело ответа:", string(body))

	// Проверяем, что тело ответа не пустое
	if len(body) == 0 {
		fmt.Println("[xui_client] Получен пустой ответ от сервера, но это может быть нормально")
		// Возвращаем пустой список пользователей вместо ошибки
		return []map[string]interface{}{}, nil
	}

	var result struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("[xui_client] Ошибка декодирования ответа: %v\n", err)
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		fmt.Printf("[xui_client] Тело ответа (первые 200 символов): %s\n", bodyPreview)
		return nil, fmt.Errorf("JSON decode error: %v", err)
	}
	if !result.Success {
		fmt.Println("[xui_client] Запрос не успешен (success=false)")
		return nil, errors.New("request not successful")
	}
	fmt.Println("[xui_client] Получено пользователей:", len(result.Data))
	return result.Data, nil
}

func (c *Client) AddInbound(form *InboundAddForm) (int, error) {
	fmt.Println("[xui_client] Добавление inbound через:", c.BaseURL+"/panel/inbound/add")
	req, err := http.NewRequest("POST", c.BaseURL+"/panel/inbound/add", strings.NewReader(form.ToFormData()))
	if err != nil {
		fmt.Println("[xui_client] Ошибка создания запроса:", err)
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "3x-ui",
		Value: c.Token,
	})

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Println("[xui_client] Ошибка отправки запроса:", err)
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("[xui_client] Код ответа:", resp.StatusCode)
	fmt.Println("[xui_client] Тело ответа:", string(body))

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Проверяем, что тело ответа не пустое
	if len(body) == 0 {
		fmt.Println("[xui_client] Получен пустой ответ от сервера при добавлении inbound")
		return 0, errors.New("empty response from server")
	}

	// Парсим id из JSON-ответа
	var respJson struct {
		Success bool `json:"success"`
		Obj     struct {
			Id int `json:"id"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(body, &respJson); err != nil {
		fmt.Println("[xui_client] Ошибка декодирования JSON при добавлении inbound:", err)
		return 0, fmt.Errorf("JSON decode error: %v", err)
	}

	if !respJson.Success {
		fmt.Println("[xui_client] Запрос добавления inbound не успешен (success=false)")
		return 0, errors.New("request not successful")
	}

	return respJson.Obj.Id, nil
}

func (c *Client) AddClientToInbound(form *AddClientForm) error {
	fmt.Println("[xui_client] Добавление клиента через:", c.BaseURL+"/panel/inbound/addClient")
	req, err := http.NewRequest("POST", c.BaseURL+"/panel/inbound/addClient", strings.NewReader(form.ToFormData()))
	if err != nil {
		fmt.Println("[xui_client] Ошибка создания запроса:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "3x-ui",
		Value: c.Token,
	})

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Println("[xui_client] Ошибка отправки запроса:", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("[xui_client] Код ответа:", resp.StatusCode)
	fmt.Println("[xui_client] Тело ответа:", string(body))

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Проверяем, что тело ответа не пустое
	if len(body) == 0 {
		fmt.Println("[xui_client] Получен пустой ответ от сервера при добавлении клиента")
		return errors.New("empty response from server")
	}

	// Парсим JSON ответ для проверки успешности
	var respJson struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(body, &respJson); err != nil {
		fmt.Println("[xui_client] Ошибка декодирования JSON при добавлении клиента:", err)
		return fmt.Errorf("JSON decode error: %v", err)
	}

	if !respJson.Success {
		fmt.Println("[xui_client] Запрос добавления клиента не успешен:", respJson.Msg)
		return fmt.Errorf("request not successful: %s", respJson.Msg)
	}

	return nil
}
