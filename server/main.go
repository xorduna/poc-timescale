package main

import (
	_ "database/sql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
)

type Config struct {
	DBURI string
}

type Server struct {
	db *sqlx.DB
}

type Metric struct {
	AssetID  uuid.UUID `gorm:"type:uuid;not null"`
	Ts       time.Time `gorm:"type:timestamptz;not null"`
	Temp     float64   `gorm:"type:float"`
	AmbHumid float64   `gorm:"type:float"`
	Setpoint float64   `gorm:"type:float"`
	Coverage float64   `gorm:"type:float"`
	AmbTemp  float64   `gorm:"type:float"`
}

type MetricRow struct {
	Ts       time.Time `json:"ts"`
	Temp     float64   `json:"temp"`
	AmbHumid float64   `json:"amb_humid"`
	Setpoint float64   `json:"setpoint"`
	AmbTemp  float64   `json:"amb_temp"`
	Coverage float64   `json:"coverage"`
}

type MetricPayload struct {
	AssetID string      `json:"asset_id"`
	Metrics []MetricRow `json:"metrics"`
}

func errorLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				log.Printf("Error: %v", err) // Registra tots els errors
				if he, ok := err.(*echo.HTTPError); ok {
					log.Printf("HTTP Error Code: %d, Message: %v", he.Code, he.Message)
				}
				return err // Retorna l'error per permetre que Echo el gestioni
			}
			return nil
		}
	}
}

func main() {
	app := &cli.App{
		Name:  "timescale-api",
		Usage: "API for Timescale POC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dburi",
				Value:   "postgres://postgres:mysecretpassword@localhost:5432/postgres?sslmode=disable", // Replace with your default value
				EnvVars: []string{"DBURI"},
				Usage:   "Database URI",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	config := Config{
		DBURI: c.String("dburi"),
	}

	db, err := sqlx.Connect("postgres", config.DBURI)
	if err != nil {
		return err
	}
	defer db.Close()

	server := &Server{db: db}

	e := echo.New()
	e.Use(errorLogger()) // Assegura't que això està abans de definir les rutes
	e.POST("/assets/:id/metrics", server.postMetrics)
	e.GET("/assets/:id/metrics", server.getMetrics)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${remote_ip} ${method} ${uri} ${status} ${latency_human}\n",
	}))

	return e.Start(":8080")
}

func (s *Server) postMetrics(c echo.Context) error {
	assetID := c.Param("id")
	var payload MetricPayload
	if err := c.Bind(&payload); err != nil {
		return err
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`
		INSERT INTO metrics (asset_id, ts, temp, amb_humid, setpoint, amb_temp, coverage)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, metric := range payload.Metrics {
		_, err := stmt.Exec(assetID, metric.Ts, metric.Temp, metric.AmbHumid, metric.Setpoint, metric.AmbTemp, metric.Coverage)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "success"})
}

func (s *Server) getMetrics(c echo.Context) error {
	assetID := c.Param("id")
	from := c.QueryParam("from")
	to := c.QueryParam("to")

	query := `
		SELECT ts, temp, amb_humid, setpoint, amb_temp
		FROM asset_metrics
		WHERE asset_id = $1 AND ts BETWEEN $2 AND $3
		ORDER BY ts
	`

	var metrics []Metric
	err := s.db.Select(&metrics, query, assetID, from, to)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, metrics)
}
