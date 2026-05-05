package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Attachment struct {
	Color string `json:"color"`
	Text  string `json:"text"`
}

type Invoice struct {
	InvoiceUUID   string `json:"invoice_uuid"`
	Amount        string `json:"amount"`
	InvoicePeriod string `json:"invoice_period"`
}

type InvoiceItem struct {
	Product string `json:"product"`
	Amount  string `json:"amount"`
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

	invoice, err := fetchLatestInvoice(token)
	if err != nil {
		fmt.Println("Error fetching invoices:", err)
		return
	}

	items, err := fetchInvoiceItems(token, invoice.InvoiceUUID)
	if err != nil {
		fmt.Println("Error fetching invoice items:", err)
		return
	}

	message := createSlackMessage(invoice, items)
	if err = postToSlack(slackURL, message); err != nil {
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

func doRequest(token, url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func fetchLatestInvoice(token string) (*Invoice, error) {
	body, err := doRequest(token, "https://api.digitalocean.com/v2/customers/my/invoices")
	if err != nil {
		return nil, err
	}

	var data struct {
		Invoices []Invoice `json:"invoices"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if len(data.Invoices) == 0 {
		return nil, fmt.Errorf("no invoices found")
	}
	return &data.Invoices[0], nil
}

func fetchInvoiceItems(token, uuid string) ([]InvoiceItem, error) {
	var allItems []InvoiceItem
	url := fmt.Sprintf("https://api.digitalocean.com/v2/customers/my/invoices/%s", uuid)

	for url != "" {
		body, err := doRequest(token, url)
		if err != nil {
			return nil, err
		}

		var data struct {
			InvoiceItems []InvoiceItem `json:"invoice_items"`
			Links        struct {
				Pages struct {
					Next string `json:"next"`
				} `json:"pages"`
			} `json:"links"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		allItems = append(allItems, data.InvoiceItems...)
		url = data.Links.Pages.Next
	}
	return allItems, nil
}

func createSlackMessage(invoice *Invoice, items []InvoiceItem) string {
	productTotals := map[string]float64{}
	for _, item := range items {
		amount, err := strconv.ParseFloat(item.Amount, 64)
		if err != nil {
			continue
		}
		productTotals[item.Product] += amount
	}

	products := make([]string, 0, len(productTotals))
	for p := range productTotals {
		products = append(products, p)
	}
	sort.Strings(products)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("https://do.co/3rn92ce\n・ %s\n・ $%s\n\n breakdown:", invoice.InvoicePeriod, invoice.Amount))
	for _, p := range products {
		sb.WriteString(fmt.Sprintf("\n  - %s: $%.2f", p, productTotals[p]))
	}
	return sb.String()
}

func postToSlack(slackURL, message string) error {
	attachment := Attachment{
		Color: "#240bde",
		Text:  message,
	}
	payload := map[string]interface{}{
		"text":        "Check billing! DigitalOcean :cloud:",
		"attachments": []Attachment{attachment},
	}

	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", slackURL, strings.NewReader(string(jsonValue)))
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
