package database

import (
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Host string
	User string
	Pass string
	Name string
	Port string
}

func loadConfig() Config {
	_ = godotenv.Load()
	return Config{
		User: os.Getenv("DB_USER"),
		Pass: os.Getenv("DB_PASSWORD"),
		Name: os.Getenv("DB_NAME"),
		Host: os.Getenv("DB_HOST"),
		Port: os.Getenv("DB_PORT")}
}

func (c Config) DSN(dbName string) string {

	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", c.Host, c.User, c.Pass, dbName, c.Port)
}

func createDatabase(cfg Config) error {
	adminDSN := cfg.DSN("postgres")
	adminDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := adminDB.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	return adminDB.Exec("CREATE DATABASE " + cfg.Name).Error
}

func ConnectAndMigrateDatabase() (*gorm.DB, error) {
	cfg := loadConfig()

	dsn := cfg.DSN(cfg.Name)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "3D000" {
			createErr := createDatabase(cfg)
			if createErr != nil {
				return nil, createErr
			}

			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return db, nil

}
