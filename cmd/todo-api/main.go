package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"api.todo.domain.com/internal/openid"
	"api.todo.domain.com/pkg/api"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	KEYCLOAK_WELL_KNOWN_ENDPOINT = "http://keycloak.api-playground.orb.local:8080/realms/myrealm/.well-known/openid-configuration"

	SERVICE_PORT = "3000"
)

func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass pre-authentication methods
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		wellKnown, err := openid.NewWellKnownConfiguration(KEYCLOAK_WELL_KNOWN_ENDPOINT)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		k, err := keyfunc.NewDefault([]string{wellKnown.JWKSUri})
		if err != nil {
			return
		}

		token, err := jwt.Parse(tokenString, k.Keyfunc)
		if err != nil || !token.Valid {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid Claims", http.StatusUnauthorized)
			return
		}

		scopes := r.Context().Value("openId.Scopes").([]string)
		httplog.LogEntrySetField(r.Context(), "authScopes", slog.StringValue(strings.Join(scopes, ",")))

		username := claims["preferred_username"].(string)
		httplog.LogEntrySetField(r.Context(), "username", slog.StringValue(username))

		next.ServeHTTP(w, r)
	})
}

func main() {
	server := api.NewTodoApiServer()

	r := chi.NewMux()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	logger := httplog.NewLogger("todo-api", httplog.Options{
		JSON:             true,
		LogLevel:         slog.LevelDebug,
		Concise:          true,
		RequestHeaders:   true,
		MessageFieldName: "message",
	})
	slog.SetDefault(logger.Logger)

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httplog.RequestLogger(logger))

	s := &http.Server{
		Addr: "0.0.0.0:" + SERVICE_PORT,
		Handler: api.HandlerWithOptions(server, api.ChiServerOptions{
			BaseRouter:  r,
			Middlewares: []api.MiddlewareFunc{jwtMiddleware},
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			},
		}),
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	slog.Info("Starting server", "port", SERVICE_PORT)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down server")

	if err := s.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown server", "error", err)
		os.Exit(1)
	}
}
