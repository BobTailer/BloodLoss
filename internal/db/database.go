package db

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	StatusDraft     = "черновик"
	StatusPublished = "опубликован"
	StatusDeleted   = "удален"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:120;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Services  []Service `gorm:"foreignKey:CreatedBy;references:ID" json:"services,omitempty"`
	Likes     []Like    `gorm:"foreignKey:UserID;references:ID" json:"likes,omitempty"`
}

type Service struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"size:200;not null" json:"name"`
	ShortDescription string     `gorm:"size:500" json:"short_description"`
	Status           string     `gorm:"size:20;not null;default:'черновик'" json:"status"`
	ImageURL         string     `gorm:"size:255" json:"image_url"`
	VideoURL         string     `gorm:"size:255" json:"video_url"`
	HbBefore         float64    `gorm:"not null;default:0" json:"hb_before"`
	HbAfter          float64    `gorm:"not null;default:0" json:"hb_after"`
	DateCreated      time.Time  `gorm:"not null" json:"date_created"`
	DateFormation    *time.Time `gorm:"default:null" json:"date_formation,omitempty"`
	CreatedBy        uint       `gorm:"not null" json:"created_by"`
	Creator          User       `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"creator,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Likes            []Like     `gorm:"foreignKey:ServiceID;references:ID" json:"likes,omitempty"`
}

type Like struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_like_user_service" json:"user_id"`
	ServiceID uint      `gorm:"not null;uniqueIndex:idx_like_user_service" json:"service_id"`
	CreatedAt time.Time `json:"created_at"`
	User      User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"user,omitempty"`
	Service   Service   `gorm:"foreignKey:ServiceID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"service,omitempty"`
}

func DSN() string {
	_ = godotenv.Load()

	if value := strings.TrimSpace(os.Getenv("DB_DSN")); value != "" {
		return value
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	if port == "" {
		port = "5432"
	}
	user := strings.TrimSpace(os.Getenv("DB_USER"))
	if user == "" {
		user = "postgres"
	}
	password := strings.TrimSpace(os.Getenv("DB_PASS"))
	if password == "" {
		password = "postgres"
	}
	name := strings.TrimSpace(os.Getenv("DB_NAME"))
	if name == "" {
		name = "bloodloss"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, name)
}

func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Service{}, &Like{}); err != nil {
		return nil, err
	}

	if err := Seed(db); err != nil {
		return nil, err
	}

	return db, nil
}

func Seed(db *gorm.DB) error {
	var user User
	if err := db.Where("id = ?", 1).FirstOrCreate(&user, User{Name: "admin"}).Error; err != nil {
		return err
	}

	var count int64
	if err := db.Model(&Service{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	now := time.Now()
	services := []Service{
		{
			Name:             "Черновик услуги",
			ShortDescription: "Черновая запись для проверки формы",
			Status:           StatusDraft,
			ImageURL:         "cesarean.svg",
			VideoURL:         "cesarean-animated.svg",
			HbBefore:         120,
			HbAfter:          95,
			DateCreated:      now,
			CreatedBy:        user.ID,
		},
		{
			Name:             "Удалённая услуга",
			ShortDescription: "Сервис, помеченный как удалённый",
			Status:           StatusDeleted,
			ImageURL:         "hip-replacement.svg",
			VideoURL:         "hip-replacement-animated.svg",
			HbBefore:         140,
			HbAfter:          110,
			DateCreated:      now,
			CreatedBy:        user.ID,
		},
		{
			Name:             "Опубликованная услуга",
			ShortDescription: "Основная опубликованная запись",
			Status:           StatusPublished,
			ImageURL:         "appendectomy.svg",
			VideoURL:         "appendectomy-animated.svg",
			HbBefore:         130,
			HbAfter:          102,
			DateCreated:      now,
			DateFormation:    &now,
			CreatedBy:        user.ID,
		},
	}

	for i := range services {
		if err := db.Create(&services[i]).Error; err != nil {
			return err
		}
	}

	return nil
}
