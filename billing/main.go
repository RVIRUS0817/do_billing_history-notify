package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Attachment struct {
	Color string `json:"color"`
	Text  string `json:"text"`
}

type BillingHistoryItem struct {
	Date   string `json:"date"`
	Amount string `json:"amount"`
}

func main() {
	token, err := getEnv("DO_TOKEN")
	if err != nil {
		fmt.Println(err)
		return
	}

	slackURL, err := getEnv("SLACK_URL")
	if err != nil {
		fmt.Println(err)
		return
	}

	billingHistory, err := fetchBillingHistory(token)
	if err != nil {
		fmt.Println("Error fetching billing history:", err)
		return
	}

	message := createSlackMessage(billingHistory)
	err = postToSlack(slackURL, message)
	if err != nil {
		fmt.Println("Error posting to Slack:", err)
	}
}

func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return value, nil
}

func fetchBillingHistory(token string) ([]BillingHistoryItem, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/customers/my/billing_history", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		BillingHistory []BillingHistoryItem `json:"billing_history"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data.BillingHistory, nil
}

func createSlackMessage(billingHistory []BillingHistoryItem) string {
	if len(billingHistory) == 0 {
		return "No billing history available"
	}
	latestBill := billingHistory[0]
	message := fmt.Sprintf("https://hoge \n ・ %s \n ・ $%s", latestBill.Date, latestBill.Amount)
	return message
}

func postToSlack(slackURL, message string) error {
	attachment := Attachment{
		Color: "#240bde",
		Text:  message,
	}
	payload := map[string]interface{}{
		"text":        "Check billing! :cloud:",
		"attachments": []Attachment{attachment},
	}

	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	reader := strings.NewReader(string(jsonValue))
	req, err := http.NewRequest("POST", slackURL, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
