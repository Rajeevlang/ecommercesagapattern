package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const gatewayURL = "http://localhost:8080/query"

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("🚀 Starting E-Commerce Saga Integration Test Runner")
	fmt.Println("==================================================")

	// 1. Wait for Gateway to be healthy
	if err := waitForGateway(30 * time.Second); err != nil {
		fmt.Printf("❌ Error: Gateway did not become healthy: %v\n", err)
		os.Exit(1)
	}

	// 2. Register Test User
	fmt.Println("\nStep 1: Registering new test user...")
	email := fmt.Sprintf("sagatester-%d@example.com", time.Now().UnixNano())
	registerQuery := fmt.Sprintf(`
		mutation {
			register(email: "%s", password: "securepassword123", name: "Saga Tester") {
				token
				user {
					id
					email
					name
				}
			}
		}
	`, email)
	resp, err := sendGraphQL(registerQuery, nil, "")
	if err != nil {
		fmt.Printf("❌ Registration failed: %v\n", err)
		os.Exit(1)
	}

	regData := resp.Data["register"].(map[string]interface{})
	token := regData["token"].(string)
	userData := regData["user"].(map[string]interface{})
	userID := userData["id"].(string)
	fmt.Printf("✅ Registered user successfully! ID: %s, Email: %s\n", userID, userData["email"])

	// 3. Query Product Catalog
	fmt.Println("\nStep 2: Querying Product Catalog...")
	catalogQuery := `
		query {
			products(first: 5) {
				edges {
					node {
						id
						name
						priceCents
						stock
					}
				}
			}
		}
	`
	resp, err = sendGraphQL(catalogQuery, nil, token)
	if err != nil {
		fmt.Printf("❌ Catalog query failed: %v\n", err)
		os.Exit(1)
	}

	prodConnection := resp.Data["products"].(map[string]interface{})
	edges := prodConnection["edges"].([]interface{})
	if len(edges) == 0 {
		fmt.Println("❌ No products found in the catalog to test order creation!")
		os.Exit(1)
	}

	firstEdge := edges[0].(map[string]interface{})
	firstNode := firstEdge["node"].(map[string]interface{})
	productID := firstNode["id"].(string)
	productName := firstNode["name"].(string)
	fmt.Printf("✅ Found product: %s (ID: %s)\n", productName, productID)

	// 4. Test Case: Successful Order Flow (Saga Completion)
	fmt.Println("\nStep 3: Creating Order (Success Path - tok_visa)...")
	createOrderMutation := `
		mutation($input: CreateOrderInput!) {
			createOrder(input: $input) {
				id
				status
				totalAmountCents
			}
		}
	`
	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"productId": productID,
					"quantity":  2,
				},
			},
			"paymentMethodToken": "tok_visa",
		},
	}

	resp, err = sendGraphQL(createOrderMutation, variables, token)
	if err != nil {
		fmt.Printf("❌ Create order failed: %v\n", err)
		os.Exit(1)
	}

	orderData := resp.Data["createOrder"].(map[string]interface{})
	orderID := orderData["id"].(string)
	initialStatus := orderData["status"].(string)
	fmt.Printf("✅ Order created! ID: %s, Initial Status: %s\n", orderID, initialStatus)

	// Poll Order status to verify it transitions to COMPLETED
	fmt.Println("Polling order status (expecting COMPLETED)...")
	completed := false
	for i := 0; i < 5; i++ {
		time.Sleep(1500 * time.Millisecond)
		status, err := queryOrderStatus(orderID, token)
		if err != nil {
			fmt.Printf("⚠️ Poll error: %v\n", err)
			continue
		}
		fmt.Printf("  - Attempt %d: Current status = %s\n", i+1, status)
		if status == "COMPLETED" {
			completed = true
			break
		}
	}

	if completed {
		fmt.Println("🎉 Success Path Verification passed! Order completed successfully.")
	} else {
		fmt.Println("❌ Success Path Verification failed: Order did not transition to COMPLETED.")
		os.Exit(1)
	}

	// 5. Test Case: Failed Order Flow (Saga Compensation rollback)
	fmt.Println("\nStep 4: Creating Order (Failure Path - tok_decline)...")
	variables["input"].(map[string]interface{})["paymentMethodToken"] = "tok_decline"

	resp, err = sendGraphQL(createOrderMutation, variables, token)
	if err != nil {
		fmt.Printf("❌ Create order failed: %v\n", err)
		os.Exit(1)
	}

	failedOrderData := resp.Data["createOrder"].(map[string]interface{})
	failedOrderID := failedOrderData["id"].(string)
	fmt.Printf("✅ Order created! ID: %s, Initial Status: %s\n", failedOrderID, failedOrderData["status"])

	// Poll Order status to verify it transitions to FAILED (due to payment decline compensation)
	fmt.Println("Polling order status (expecting FAILED)...")
	failed := false
	for i := 0; i < 5; i++ {
		time.Sleep(1500 * time.Millisecond)
		status, err := queryOrderStatus(failedOrderID, token)
		if err != nil {
			fmt.Printf("⚠️ Poll error: %v\n", err)
			continue
		}
		fmt.Printf("  - Attempt %d: Current status = %s\n", i+1, status)
		if status == "FAILED" {
			failed = true
			break
		}
	}

	if failed {
		fmt.Println("🎉 Failure Path Verification passed! Order rolled back and marked FAILED.")
	} else {
		fmt.Println("❌ Failure Path Verification failed: Order did not transition to FAILED.")
		os.Exit(1)
	}

	fmt.Println("\n==================================================")
	fmt.Println("🏆 Integration Tests finished successfully!")
	fmt.Println("==================================================")
}

func waitForGateway(timeout time.Duration) error {
	fmt.Println("Waiting for GraphQL API Gateway to become reachable...")
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get("http://localhost:8080/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				fmt.Println("GraphQL API Gateway is online and ready.")
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout waiting for gateway")
}

func queryOrderStatus(orderID, token string) (string, error) {
	query := fmt.Sprintf(`
		query {
			order(id: "%s") {
				status
			}
		}
	`, orderID)

	resp, err := sendGraphQL(query, nil, token)
	if err != nil {
		return "", err
	}

	orderData := resp.Data["order"].(map[string]interface{})
	return orderData["status"].(string), nil
}

func sendGraphQL(query string, variables map[string]interface{}, token string) (*graphQLResponse, error) {
	reqBody, err := json.Marshal(graphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return &gqlResp, nil
}
