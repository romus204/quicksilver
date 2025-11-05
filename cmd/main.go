package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/romus204/quicksilver/internal/solver/greedy"
)

func main() {
	app := fiber.New()

	app.Post("/vpr/greedy", func(c *fiber.Ctx) error {
		var req greedy.Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		resp := greedy.SolveVPR(req)
		return c.JSON(resp)
	})

	app.Listen(":3000")
}
