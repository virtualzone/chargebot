package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type TelegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func SendPushNotification(msg string) error {
	if GetConfig().TelegramToken != "" {
		return sendTelegramMessage(msg)
	}
	return nil
}

func sendTelegramMessage(msg string) error {
	payload := TelegramMessage{
		ChatID: GetConfig().TelegramChatID,
		Text:   msg,
	}
	json, _ := json.Marshal(payload)
	target := "https://api.telegram.org/bot" + GetConfig().TelegramToken + "/sendMessage"
	r, _ := http.NewRequest("POST", target, bytes.NewReader(json))
	resp, err := RetryHTTPJSONRequest(r, "")

	if err != nil {
		return err
	}

	var m TelegramResponse
	if err := UnmarshalBody(resp.Body, &m); err != nil {
		return err
	}

	if !m.OK {
		return fmt.Errorf("could not send message to telegram: error code %d (%s)", m.ErrorCode, m.Description)
	}

	return nil
}
