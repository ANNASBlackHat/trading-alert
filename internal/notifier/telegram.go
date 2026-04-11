package notifier

import (
	"fmt"
	"net/http"
	"net/url"
)

type TelegramNotifier struct {
	Token  string
	ChatID string
}

func NewTelegramNotifier(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		Token:  token,
		ChatID: chatID,
	}
}

func (t *TelegramNotifier) Send(text string) error {
	if t.Token == "" || t.ChatID == "" {
		fmt.Printf("📨 TELEGRAM (console only): %s\n\n", text)
		return nil
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&parse_mode=HTML&text=%s",
		t.Token, t.ChatID, url.QueryEscape(text))
	
	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from telegram: %d", resp.StatusCode)
	}

	return nil
}
