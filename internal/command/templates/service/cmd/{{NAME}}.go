package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"golang.org/x/oauth2"

	auth "github.com/mrshanahan/quemot-dev-auth-client/pkg/fiber"
)

func main() {
	exitCode := Run()
	os.Exit(exitCode)
}

const (
	TokenCookieName string = "access_token"
	TokenLocalName  string = "access_token"
	DefaultPort     int    = {{API_PORT_DEFAULT}}
)

func Run() int {
	portStr := os.Getenv("{{API_PORT_ENVVAR}}")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = DefaultPort
		slog.Info("no valid port provided via {{API_PORT_ENVVAR}}, using default",
			"portStr", portStr,
			"defaultPort", port)
	} else {
		slog.Info("using custom port",
			"port", port)
	}

	allowedOrigins := "*" // TODO: Change me!
	app := fiber.New()
	app.Use(requestid.New(), logger.New(), recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
	}))
	app.Route("/foo", func(foo fiber.Router) {
		foo.Use(auth.ValidateAccessTokenMiddleware(TokenLocalName, TokenCookieName))
		foo.Get("/", func(c *fiber.Ctx) error {
			slog.Info("Getting foo'd")
			return c.SendString("OK")
		})
	})
	app.Route("/auth", func(authn fiber.Router) {
		authn.Get("/login", auth.NewLoginController(auth.OriginStateFactory("came_from")))
		authn.Get("/logout", func(c *fiber.Ctx) error {
			// TODO: Invalidate token(s)
			c.ClearCookie(TokenCookieName)
			return c.SendString("Logout successful")
		})
		authn.Get("/callback", auth.NewCallbackController(func(c *fiber.Ctx, s auth.OriginState, t *oauth2.Token) error {
			c.Cookie(&fiber.Cookie{
				Name:  TokenCookieName,
				Value: t.AccessToken,
			})

			if s.CameFrom != "" {
				c.Redirect(s.CameFrom)
			}
			return c.SendString("Login successful")
		}))
	})

	slog.Info("listening for requests", "port", port)
	err = app.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		slog.Error("failed to initialize HTTP server",
			"err", err)
		return 1
	}
	return 0
}
