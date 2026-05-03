package server

import "github.com/gofiber/fiber/v3"

func Run() error {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	return app.Listen(":3000")
}
