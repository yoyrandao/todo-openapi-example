package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"api.todo.domain.com/internal/openid"
	"api.todo.domain.com/pkg/client"
)

const (
	KEYCLOAK_WELL_KNOWN_ENDPOINT = "http://keycloak.api-playground.orb.local:8080/realms/myrealm/.well-known/openid-configuration"

	SERVER_URL = "http://localhost:3000"
)

var (
	KEYCLOAK_CLIENT_ID     = os.Getenv("KEYCLOAK_CLIENT_ID")
	KEYCLOAK_CLIENT_SECRET = os.Getenv("KEYCLOAK_CLIENT_SECRET")
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func getAccessToken(httpClient *http.Client) (string, error) {
	wellKnown, err := openid.NewWellKnownConfiguration(KEYCLOAK_WELL_KNOWN_ENDPOINT)
	if err != nil {
		return "", err
	}

	data := []byte("client_id=" + KEYCLOAK_CLIENT_ID + "&client_secret=" + KEYCLOAK_CLIENT_SECRET + "&grant_type=client_credentials")

	request, err := http.NewRequest("POST", wellKnown.TokenEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", response.Status)
	}

	var tokenResponse TokenResponse

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

func main() {
	httpClient := http.Client{}

	token, err := getAccessToken(&httpClient)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := client.NewClient(
		SERVER_URL,
		client.WithHTTPClient(&httpClient),
		client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}

	response, err := c.AddTodo(ctx, client.AddTodoJSONRequestBody{
		Name:        "test-todo",
		Description: "test",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("add-todo: " + fmt.Sprint(response.StatusCode))

	response, err = c.GetTodos(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("get-todos: " + fmt.Sprint(response.StatusCode))

	var tasks []client.Todo

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&tasks); err != nil {
		panic(err)
	}

	for _, task := range tasks {
		fmt.Printf("%s: %s\n", task.Name, task.Description)
	}
}
