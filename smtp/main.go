package smtp

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	APIUrl = "https://api.sendpulse.com"
)

// Error messages
const (
	ErrInvalidToken       = "Invalid token"
	ErrInvalidResponse    = "Bad response from server"
	ErrInvalidCredentials = "Invalid credentials"
)

// Client represents the SendPulse API client
type Client struct {
	UserID       string
	Secret       string
	TokenStorage string
	Token        string
	httpClient   *http.Client
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	IsError   bool   `json:"is_error"`
	Message   string `json:"message,omitempty"`
	ErrorCode int    `json:"error_code,omitempty"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// AddressBook represents an address book
type AddressBook struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Email represents an email address with variables
type Email struct {
	Email     string                 `json:"email"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// Campaign represents an email campaign
type Campaign struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	SenderName  string `json:"sender_name"`
	SenderEmail string `json:"sender_email"`
	Subject     string `json:"subject"`
}

// SMSCampaign represents an SMS campaign
type SMSCampaign struct {
	ID     int    `json:"id"`
	Sender string `json:"sender"`
	Body   string `json:"body"`
	Status string `json:"status"`
}

// Phone represents a phone number with variables
type Phone struct {
	Phone     string                 `json:"phone"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// NewClient creates a new SendPulse API client
func NewClient(userID, secret, tokenStorage string) *Client {
	return &Client{
		UserID:       userID,
		Secret:       secret,
		TokenStorage: tokenStorage,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Init initializes the client and loads/retrieves the access token
func (c *Client) Init() error {
	// Create token storage directory if it doesn't exist
	if err := os.MkdirAll(c.TokenStorage, 0755); err != nil {
		return fmt.Errorf("failed to create token storage directory: %w", err)
	}

	// Generate hash for token filename
	hashName := fmt.Sprintf("%x", md5.Sum([]byte(c.UserID+"::"+c.Secret)))
	tokenPath := filepath.Join(c.TokenStorage, hashName)

	// Try to load existing token
	if tokenData, err := os.ReadFile(tokenPath); err == nil {
		c.Token = string(tokenData)
	}

	// If no token or token is empty, get a new one
	if c.Token == "" {
		return c.getToken()
	}

	return nil
}

// getToken retrieves a new access token from the API
func (c *Client) getToken() error {
	data := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     c.UserID,
		"client_secret": c.Secret,
	}

	resp, err := c.sendRequest("oauth/access_token", "POST", data, false)
	if err != nil {
		return err
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(resp, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.Token = tokenResp.AccessToken

	// Save token to file
	hashName := fmt.Sprintf("%x", md5.Sum([]byte(c.UserID+"::"+c.Secret)))
	tokenPath := filepath.Join(c.TokenStorage, hashName)
	return os.WriteFile(tokenPath, []byte(c.Token), 0644)
}

// sendRequest sends an HTTP request to the API
func (c *Client) sendRequest(path, method string, data interface{}, useToken bool) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", APIUrl, path)

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if useToken && c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle 401 Unauthorized - token might be expired
	if resp.StatusCode == 401 {
		var errResp ErrorResponse
		json.Unmarshal(respBody, &errResp)

		if strings.Contains(string(respBody), "invalid_client") {
			return nil, fmt.Errorf(ErrInvalidCredentials)
		}

		// Try to refresh token and retry request
		if err := c.getToken(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Retry the request with new token
		return c.sendRequest(path, method, data, true)
	}

	return respBody, nil
}

// Address Books

// ListAddressBooks retrieves the list of address books
func (c *Client) ListAddressBooks(limit, offset int) ([]AddressBook, error) {
	params := make(map[string]interface{})
	if limit > 0 {
		params["limit"] = limit
	}
	if offset > 0 {
		params["offset"] = offset
	}

	resp, err := c.sendRequest("addressbooks", "GET", params, true)
	if err != nil {
		return nil, err
	}

	var books []AddressBook
	if err := json.Unmarshal(resp, &books); err != nil {
		return nil, fmt.Errorf("failed to parse address books: %w", err)
	}

	return books, nil
}

// CreateAddressBook creates a new address book
func (c *Client) CreateAddressBook(name string) (*AddressBook, error) {
	if name == "" {
		return nil, fmt.Errorf("empty book name")
	}

	data := map[string]string{"bookName": name}
	resp, err := c.sendRequest("addressbooks", "POST", data, true)
	if err != nil {
		return nil, err
	}

	var book AddressBook
	if err := json.Unmarshal(resp, &book); err != nil {
		return nil, fmt.Errorf("failed to parse address book: %w", err)
	}

	return &book, nil
}

// EditAddressBook edits an address book name
func (c *Client) EditAddressBook(id int, name string) error {
	if id == 0 || name == "" {
		return fmt.Errorf("empty book name or book id")
	}

	data := map[string]string{"name": name}
	_, err := c.sendRequest(fmt.Sprintf("addressbooks/%d", id), "PUT", data, true)
	return err
}

// RemoveAddressBook removes an address book
func (c *Client) RemoveAddressBook(id int) error {
	if id == 0 {
		return fmt.Errorf("empty book id")
	}

	_, err := c.sendRequest(fmt.Sprintf("addressbooks/%d", id), "DELETE", nil, true)
	return err
}

// GetBookInfo retrieves information about an address book
func (c *Client) GetBookInfo(id int) (*AddressBook, error) {
	if id == 0 {
		return nil, fmt.Errorf("empty book id")
	}

	resp, err := c.sendRequest(fmt.Sprintf("addressbooks/%d", id), "GET", nil, true)
	if err != nil {
		return nil, err
	}

	var book AddressBook
	if err := json.Unmarshal(resp, &book); err != nil {
		return nil, fmt.Errorf("failed to parse address book: %w", err)
	}

	return &book, nil
}

// Email Management

// GetEmailsFromBook retrieves email addresses from an address book
func (c *Client) GetEmailsFromBook(id int) ([]Email, error) {
	if id == 0 {
		return nil, fmt.Errorf("empty book id")
	}

	resp, err := c.sendRequest(fmt.Sprintf("addressbooks/%d/emails", id), "GET", nil, true)
	if err != nil {
		return nil, err
	}

	var emails []Email
	if err := json.Unmarshal(resp, &emails); err != nil {
		return nil, fmt.Errorf("failed to parse emails: %w", err)
	}

	return emails, nil
}

// AddEmails adds new emails to an address book
func (c *Client) AddEmails(bookID int, emails []Email) error {
	if bookID == 0 || len(emails) == 0 {
		return fmt.Errorf("empty email list or book id")
	}

	emailsJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to serialize emails: %w", err)
	}

	data := map[string]string{"emails": string(emailsJSON)}
	_, err = c.sendRequest(fmt.Sprintf("addressbooks/%d/emails", bookID), "POST", data, true)
	return err
}

// RemoveEmails removes email addresses from an address book
func (c *Client) RemoveEmails(bookID int, emails []string) error {
	if bookID == 0 || len(emails) == 0 {
		return fmt.Errorf("empty email list or book id")
	}

	emailsJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to serialize emails: %w", err)
	}

	data := map[string]string{"emails": string(emailsJSON)}
	_, err = c.sendRequest(fmt.Sprintf("addressbooks/%d/emails", bookID), "DELETE", data, true)
	return err
}

// GetEmailInfo retrieves information about an email address from an address book
func (c *Client) GetEmailInfo(bookID int, email string) (*Email, error) {
	if bookID == 0 || email == "" {
		return nil, fmt.Errorf("empty email or book id")
	}

	resp, err := c.sendRequest(fmt.Sprintf("addressbooks/%d/emails/%s", bookID, email), "GET", nil, true)
	if err != nil {
		return nil, err
	}

	var emailInfo Email
	if err := json.Unmarshal(resp, &emailInfo); err != nil {
		return nil, fmt.Errorf("failed to parse email info: %w", err)
	}

	return &emailInfo, nil
}

// UpdateEmailVariables updates variables for an email address in an address book
func (c *Client) UpdateEmailVariables(bookID int, email string, variables map[string]interface{}) error {
	if bookID == 0 || email == "" || len(variables) == 0 {
		return fmt.Errorf("empty email, variables or book id")
	}

	data := map[string]interface{}{
		"email":     email,
		"variables": variables,
	}

	_, err := c.sendRequest(fmt.Sprintf("addressbooks/%d/emails/variable", bookID), "POST", data, true)
	return err
}

// Campaigns

// ListCampaigns retrieves the list of campaigns
func (c *Client) ListCampaigns(limit, offset int) ([]Campaign, error) {
	params := make(map[string]interface{})
	if limit > 0 {
		params["limit"] = limit
	}
	if offset > 0 {
		params["offset"] = offset
	}

	resp, err := c.sendRequest("campaigns", "GET", params, true)
	if err != nil {
		return nil, err
	}

	var campaigns []Campaign
	if err := json.Unmarshal(resp, &campaigns); err != nil {
		return nil, fmt.Errorf("failed to parse campaigns: %w", err)
	}

	return campaigns, nil
}

// GetCampaignInfo retrieves information about a campaign
func (c *Client) GetCampaignInfo(id int) (*Campaign, error) {
	if id == 0 {
		return nil, fmt.Errorf("empty campaign id")
	}

	resp, err := c.sendRequest(fmt.Sprintf("campaigns/%d", id), "GET", nil, true)
	if err != nil {
		return nil, err
	}

	var campaign Campaign
	if err := json.Unmarshal(resp, &campaign); err != nil {
		return nil, fmt.Errorf("failed to parse campaign: %w", err)
	}

	return &campaign, nil
}

// CreateCampaign creates a new email campaign
func (c *Client) CreateCampaign(senderName, senderEmail, subject, body string, bookID int, name string, attachments []string) (*Campaign, error) {
	if senderName == "" || senderEmail == "" || subject == "" || body == "" || bookID == 0 {
		return nil, fmt.Errorf("missing required campaign data")
	}

	data := map[string]interface{}{
		"sender_name":  senderName,
		"sender_email": senderEmail,
		"subject":      subject,
		"body":         base64.StdEncoding.EncodeToString([]byte(body)),
		"list_id":      bookID,
		"name":         name,
	}

	if len(attachments) > 0 {
		attachmentsJSON, _ := json.Marshal(attachments)
		data["attachments"] = string(attachmentsJSON)
	}

	resp, err := c.sendRequest("campaigns", "POST", data, true)
	if err != nil {
		return nil, err
	}

	var campaign Campaign
	if err := json.Unmarshal(resp, &campaign); err != nil {
		return nil, fmt.Errorf("failed to parse campaign: %w", err)
	}

	return &campaign, nil
}

// CancelCampaign cancels a campaign
func (c *Client) CancelCampaign(id int) error {
	if id == 0 {
		return fmt.Errorf("empty campaign id")
	}

	_, err := c.sendRequest(fmt.Sprintf("campaigns/%d", id), "DELETE", nil, true)
	return err
}

// SMTP Functions

// SMTPSendMail sends an email via SMTP
func (c *Client) SMTPSendMail(emailData map[string]interface{}) error {
	if emailData == nil {
		return fmt.Errorf("empty email data")
	}

	// Encode HTML content if present
	if html, ok := emailData["html"].(string); ok {
		emailData["html"] = base64.StdEncoding.EncodeToString([]byte(html))
	}

	emailJSON, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("failed to serialize email data: %w", err)
	}

	data := map[string]string{"email": string(emailJSON)}
	s, err := c.sendRequest("smtp/emails", "POST", data, true)
	fmt.Printf("Response: %s\n", string(s))
	return err
}

// SMTPListEmails retrieves list of sent emails
func (c *Client) SMTPListEmails(limit, offset int, fromDate, toDate, sender, recipient string) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"limit":     limit,
		"offset":    offset,
		"from":      fromDate,
		"to":        toDate,
		"sender":    sender,
		"recipient": recipient,
	}

	resp, err := c.sendRequest("smtp/emails", "GET", params, true)
	if err != nil {
		return nil, err
	}

	var emails []map[string]interface{}
	if err := json.Unmarshal(resp, &emails); err != nil {
		return nil, fmt.Errorf("failed to parse emails: %w", err)
	}

	return emails, nil
}

// SMS Functions

// SMSAddPhones adds phone numbers to an address book
func (c *Client) SMSAddPhones(bookID int, phones []string) error {
	if bookID == 0 || len(phones) == 0 {
		return fmt.Errorf("empty phones or book id")
	}

	phonesJSON, err := json.Marshal(phones)
	if err != nil {
		return fmt.Errorf("failed to serialize phones: %w", err)
	}

	data := map[string]interface{}{
		"addressBookId": bookID,
		"phones":        string(phonesJSON),
	}

	_, err = c.sendRequest("sms/numbers", "POST", data, true)
	return err
}

// SMSAddPhonesWithVariables adds phone numbers with variables to an address book
func (c *Client) SMSAddPhonesWithVariables(bookID int, phones []Phone) error {
	if bookID == 0 || len(phones) == 0 {
		return fmt.Errorf("empty phones or book id")
	}

	phonesJSON, err := json.Marshal(phones)
	if err != nil {
		return fmt.Errorf("failed to serialize phones: %w", err)
	}

	data := map[string]interface{}{
		"addressBookId": bookID,
		"phones":        string(phonesJSON),
	}

	_, err = c.sendRequest("sms/numbers/variables", "POST", data, true)
	return err
}

// SMSSend sends SMS to specified phone numbers
func (c *Client) SMSSend(senderName string, phones []string, body string, date *time.Time, transliterate bool, route string) error {
	if senderName == "" || len(phones) == 0 || body == "" {
		return fmt.Errorf("missing required SMS data")
	}

	phonesJSON, err := json.Marshal(phones)
	if err != nil {
		return fmt.Errorf("failed to serialize phones: %w", err)
	}

	data := map[string]interface{}{
		"sender":        senderName,
		"phones":        string(phonesJSON),
		"body":          body,
		"transliterate": transliterate,
		"route":         route,
	}

	if date != nil {
		data["date"] = date.Format("2006-01-02 15:04:05")
	}

	_, err = c.sendRequest("sms/send", "POST", data, true)
	return err
}

// SMSAddCampaign creates a new SMS campaign
func (c *Client) SMSAddCampaign(senderName string, bookID int, body string, date *time.Time, transliterate bool) (*SMSCampaign, error) {
	if senderName == "" || bookID == 0 || body == "" {
		return nil, fmt.Errorf("missing required SMS campaign data")
	}

	data := map[string]interface{}{
		"sender":        senderName,
		"addressBookId": bookID,
		"body":          body,
		"transliterate": transliterate,
	}

	if date != nil {
		data["date"] = date.Format("2006-01-02 15:04:05")
	}

	resp, err := c.sendRequest("sms/campaigns", "POST", data, true)
	if err != nil {
		return nil, err
	}

	var campaign SMSCampaign
	if err := json.Unmarshal(resp, &campaign); err != nil {
		return nil, fmt.Errorf("failed to parse SMS campaign: %w", err)
	}

	return &campaign, nil
}

// Utility Functions

// GetBalance retrieves account balance
func (c *Client) GetBalance(currency string) (map[string]interface{}, error) {
	url := "balance"
	if currency != "" {
		url = fmt.Sprintf("balance/%s", strings.ToUpper(currency))
	}

	resp, err := c.sendRequest(url, "GET", nil, true)
	if err != nil {
		return nil, err
	}

	var balance map[string]interface{}
	if err := json.Unmarshal(resp, &balance); err != nil {
		return nil, fmt.Errorf("failed to parse balance: %w", err)
	}

	return balance, nil
}

// SendRawRequest sends a raw request to the API
func (c *Client) SendRawRequest(path, method string, data interface{}) ([]byte, error) {
	allowedMethods := []string{"POST", "GET", "DELETE", "PUT", "PATCH"}
	methodAllowed := false
	for _, m := range allowedMethods {
		if method == m {
			methodAllowed = true
			break
		}
	}

	if !methodAllowed {
		return nil, fmt.Errorf("method not allowed")
	}

	return c.sendRequest(path, method, data, true)
}
