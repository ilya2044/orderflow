package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const baseURL = "http://localhost:8080/api/v1"

type AuthResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

var client = &http.Client{Timeout: 10 * time.Second}

func main() {
	fmt.Println("🌱 Seeding database...")

	accessToken := loginAdmin()

	fmt.Println("📦 Creating products...")
	createProducts(accessToken)

	fmt.Println("🛒 Creating sample orders...")
	createOrders(accessToken)

	fmt.Println("✅ Seed completed successfully!")
}

func loginAdmin() string {
	payload := map[string]string{
		"email":    "admin@order.dev",
		"password": "admin123",
	}

	body, _ := json.Marshal(payload)
	resp, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var authResp AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	return authResp.Data.AccessToken
}

func createProducts(token string) {
	products := []map[string]interface{}{
		{
			"name":        "iPhone 15 Pro Max",
			"description": "Смартфон Apple с процессором A17 Pro, 48 МП камерой и титановым корпусом",
			"price":       129990,
			"category":    "Электроника",
			"sku":         "APPLE-IP15PM-256",
			"stock":       50,
			"tags":        []string{"apple", "смартфон", "iphone"},
		},
		{
			"name":        "MacBook Pro 14\" M3",
			"description": "Ноутбук Apple с чипом M3, 16 ГБ RAM, 512 ГБ SSD",
			"price":       199990,
			"category":    "Электроника",
			"sku":         "APPLE-MBP14-M3-512",
			"stock":       25,
			"tags":        []string{"apple", "ноутбук", "macbook"},
		},
		{
			"name":        "AirPods Pro 2",
			"description": "Беспроводные наушники с активным шумоподавлением и адаптивным аудио",
			"price":       24990,
			"category":    "Электроника",
			"sku":         "APPLE-APP2-USB-C",
			"stock":       100,
			"tags":        []string{"apple", "наушники", "airpods"},
		},
		{
			"name":        "Nike Air Max 270",
			"description": "Кроссовки Nike с технологией Air Max для максимального комфорта",
			"price":       12990,
			"category":    "Одежда",
			"sku":         "NIKE-AM270-BLK-42",
			"stock":       75,
			"tags":        []string{"nike", "кроссовки", "спорт"},
		},
		{
			"name":        "Чистый код. Роберт Мартин",
			"description": "Практическое руководство по написанию чистого, поддерживаемого кода",
			"price":       1490,
			"category":    "Книги",
			"sku":         "BOOK-CLEAN-CODE-RU",
			"stock":       200,
			"tags":        []string{"программирование", "книги", "clean code"},
		},
		{
			"name":        "Samsung Galaxy S24 Ultra",
			"description": "Флагманский смартфон Samsung со встроенным стилусом S Pen и AI функциями",
			"price":       109990,
			"category":    "Электроника",
			"sku":         "SAMSUNG-S24U-256-BLK",
			"stock":       40,
			"tags":        []string{"samsung", "смартфон", "android"},
		},
		{
			"name":        "Xiaomi Mi Robot Vacuum S10+",
			"description": "Робот-пылесос с лазерной навигацией, 3000 Па всасывания и самоочисткой",
			"price":       34990,
			"category":    "Дом и сад",
			"sku":         "XIAOMI-RV-S10PLUS",
			"stock":       30,
			"tags":        []string{"xiaomi", "пылесос", "умный дом"},
		},
		{
			"name":        "Набор гантелей 2x20 кг",
			"description": "Разборные гантели для домашних тренировок, сталь + резиновые диски",
			"price":       8990,
			"category":    "Спорт",
			"sku":         "SPORT-DUMBBELL-2X20",
			"stock":       60,
			"tags":        []string{"спорт", "гантели", "фитнес"},
		},
	}

	for _, p := range products {
		body, _ := json.Marshal(p)
		req, _ := http.NewRequest("POST", baseURL+"/products", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create product failed: %v\n", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			fmt.Printf("  ✓ Created: %s\n", p["name"])
		} else {
			fmt.Printf("  ✗ Failed: %s (status %d)\n", p["name"], resp.StatusCode)
		}
	}
}

func createOrders(token string) {
	orderReq := map[string]interface{}{
		"shipping_address": "г. Москва, ул. Арбат, д. 1, кв. 42",
		"notes":            "Позвонить за 30 минут до доставки",
		"items": []map[string]interface{}{
			{
				"product_id": "placeholder",
				"name":       "iPhone 15 Pro Max",
				"price":      129990.0,
				"quantity":   1,
			},
			{
				"product_id": "placeholder2",
				"name":       "AirPods Pro 2",
				"price":      24990.0,
				"quantity":   1,
			},
		},
	}

	body, _ := json.Marshal(orderReq)
	req, _ := http.NewRequest("POST", baseURL+"/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create order failed: %v\n", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		fmt.Println("  ✓ Created sample order")
	}
}
