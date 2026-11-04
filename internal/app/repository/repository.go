package repository

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"BloodLoss/internal/db"

	"gorm.io/gorm"
)

type Operation struct {
	ID          int
	Name        string
	Description string
	Avatar      string
	Photo       string
	Video       string
	HbBefore    float64
	HbAfter     float64
	Status      string
	CreatedBy   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Order struct {
	ID        int
	Operation Operation
	Result    string
}

type Repository struct {
	DB                  *gorm.DB
	SelectedOperationID int
}

func NewRepository() *Repository {
	repo := &Repository{SelectedOperationID: 1}
	gormDB, err := db.Connect()
	if err != nil {
		log.Printf("database unavailable, using fallback: %v", err)
		return repo
	}
	repo.DB = gormDB
	return repo
}

func fallbackOperations() []Operation {
	return []Operation{
		{ID: 1, Name: "Кесарево сечение", Description: "Оперативное родоразрешение", Avatar: "cesarean-animated.svg", Photo: "cesarean.svg", Video: "cesarean-animated.svg", HbBefore: 120, HbAfter: 95, Status: db.StatusPublished},
		{ID: 2, Name: "Эндопротезирование тазобедренного сустава", Description: "Замена тазобедренного сустава эндопротезом", Avatar: "hip-replacement-animated.svg", Photo: "hip-replacement.svg", Video: "hip-replacement-animated.svg", HbBefore: 130, HbAfter: 100, Status: db.StatusPublished},
		{ID: 3, Name: "Спондилодез", Description: "Фиксация позвонков для стабилизации позвоночника", Avatar: "spinal-fusion-animated.svg", Photo: "spinal-fusion.svg", Video: "spinal-fusion-animated.svg", HbBefore: 118, HbAfter: 97, Status: db.StatusPublished},
		{ID: 4, Name: "Аппендэктомия", Description: "Удаление червеобразного отростка", Avatar: "appendectomy-animated.svg", Photo: "appendectomy.svg", Video: "appendectomy-animated.svg", HbBefore: 126, HbAfter: 104, Status: db.StatusPublished},
		{ID: 5, Name: "Холецистэктомия", Description: "Удаление желчного пузыря", Avatar: "cholecystectomy-animated.svg", Photo: "cholecystectomy.svg", Video: "cholecystectomy-animated.svg", HbBefore: 123, HbAfter: 99, Status: db.StatusPublished},
		{ID: 6, Name: "Резекция кишечника", Description: "Удаление пораженного участка кишечника", Avatar: "bowel-resection-animated.svg", Photo: "bowel-resection.svg", Video: "bowel-resection-animated.svg", HbBefore: 128, HbAfter: 106, Status: db.StatusPublished},
	}
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "cesarean.svg"
	}
	return raw
}

func (r *Repository) serviceToOperation(service db.Service) Operation {
	avatar := normalizeURL(service.VideoURL)
	if avatar == "" {
		avatar = "cesarean-animated.svg"
	}
	photo := normalizeURL(service.ImageURL)
	if photo == "" {
		photo = "cesarean.svg"
	}
	video := normalizeURL(service.VideoURL)
	if video == "" {
		video = "cesarean-animated.svg"
	}

	return Operation{
		ID:          int(service.ID),
		Name:        service.Name,
		Description: service.ShortDescription,
		Avatar:      avatar,
		Photo:       photo,
		Video:       video,
		HbBefore:    service.HbBefore,
		HbAfter:     service.HbAfter,
		Status:      service.Status,
		CreatedBy:   int(service.CreatedBy),
		CreatedAt:   service.DateCreated,
		UpdatedAt:   service.UpdatedAt,
	}
}

func (r *Repository) GetOperations() []Operation {
	if r.DB != nil {
		var services []db.Service
		if err := r.DB.Where("status = ?", db.StatusPublished).Find(&services).Error; err == nil {
			operations := make([]Operation, 0, len(services))
			for _, service := range services {
				operations = append(operations, r.serviceToOperation(service))
			}
			return operations
		}
	}
	return fallbackOperations()
}

func (r *Repository) GetOperationByID(id int) (Operation, bool) {
	if r.DB != nil {
		var service db.Service
		if err := r.DB.First(&service, id).Error; err == nil {
			return r.serviceToOperation(service), true
		}
	}

	for _, operation := range fallbackOperations() {
		if operation.ID == id {
			return operation, true
		}
	}
	return Operation{}, false
}

func (r *Repository) FilterOperations(query string) []Operation {
	if r.DB != nil {
		var services []db.Service
		search := strings.TrimSpace(query)
		q := r.DB.Where("status = ?", db.StatusPublished)
		if search != "" {
			q = q.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(search)+"%")
		}
		if err := q.Order("date_created DESC").Find(&services).Error; err == nil {
			operations := make([]Operation, 0, len(services))
			for _, service := range services {
				operations = append(operations, r.serviceToOperation(service))
			}
			return operations
		}
	}

	if query == "" {
		return r.GetOperations()
	}

	result := make([]Operation, 0)
	for _, operation := range r.GetOperations() {
		if strings.Contains(strings.ToLower(operation.Name), strings.ToLower(query)) {
			result = append(result, operation)
		}
	}
	return result
}

func (r *Repository) GetOrder(operationID int) Order {
	operation, ok := r.GetOperationByID(operationID)
	if !ok {
		operation = fallbackOperations()[0]
	}
	return Order{ID: operationID, Operation: operation, Result: "1000 мл"}
}

func (r *Repository) SelectOperation(id int) {
	r.SelectedOperationID = id
}

func (r *Repository) GetDraftByUser(userID int) (*db.Service, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}

	var service db.Service
	if err := r.DB.Where("created_by = ? AND status = ?", userID, db.StatusDraft).First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *Repository) CreateDraft(userID int, name string, description string, imageURL string, videoURL string, hbBefore float64, hbAfter float64) (*db.Service, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}

	service, err := r.GetDraftByUser(userID)
	if err != nil {
		return nil, err
	}
	if service != nil {
		service.Name = name
		service.ShortDescription = description
		service.ImageURL = normalizeURL(imageURL)
		service.VideoURL = normalizeURL(videoURL)
		service.HbBefore = hbBefore
		service.HbAfter = hbAfter
		if err := r.DB.Save(service).Error; err != nil {
			return nil, err
		}
		return service, nil
	}

	service = &db.Service{
		Name:             name,
		ShortDescription: description,
		ImageURL:         normalizeURL(imageURL),
		VideoURL:         normalizeURL(videoURL),
		Status:           db.StatusDraft,
		HbBefore:         hbBefore,
		HbAfter:          hbAfter,
		DateCreated:      time.Now(),
		CreatedBy:        uint(userID),
	}
	if err := r.DB.Create(service).Error; err != nil {
		return nil, err
	}
	return service, nil
}

func (r *Repository) PublishService(id int, description string, hbBefore float64, hbAfter float64) error {
	if r.DB == nil {
		return errors.New("database is not available")
	}

	now := time.Now()
	result := r.DB.Model(&db.Service{}).Where("id = ?", id).Updates(map[string]interface{}{
		"short_description": description,
		"hb_before":         hbBefore,
		"hb_after":          hbAfter,
		"status":            db.StatusPublished,
		"date_formation":    now,
	})
	return result.Error
}

func (r *Repository) DeleteService(id int) error {
	if r.DB == nil {
		return errors.New("database is not available")
	}

	result := r.DB.Exec("UPDATE services SET status = ? WHERE id = ?", db.StatusDeleted, id)
	return result.Error
}

func (r *Repository) GetOnePublishedService() (*db.Service, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}

	var service db.Service
	if err := r.DB.Where("status = ?", db.StatusPublished).Order("date_created DESC").First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &service, nil
}

func (r *Repository) GetPublishedServices() ([]Operation, error) {
	if r.DB == nil {
		return r.GetOperations(), nil
	}
	var services []db.Service
	if err := r.DB.Where("status = ?", db.StatusPublished).Order("date_created DESC").Find(&services).Error; err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, len(services))
	for _, service := range services {
		operations = append(operations, r.serviceToOperation(service))
	}
	return operations, nil
}

func (r *Repository) CountLikesByService(serviceID int) (int64, error) {
	if r.DB == nil {
		return 0, nil
	}
	var count int64
	if err := r.DB.Model(&db.Like{}).Where("service_id = ?", serviceID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) FindServiceByID(id int) (*db.Service, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}
	var service db.Service
	if err := r.DB.First(&service, id).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

func (r *Repository) GetFirstDraftByUser(userID int) (*db.Service, error) {
	return r.GetDraftByUser(userID)
}

func (r *Repository) GetOperationsByMinHb(minHb float64) ([]Operation, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}
	var services []db.Service
	if err := r.DB.Where("status = ? AND hb_before >= ?", db.StatusPublished, minHb).Order("date_created DESC").Find(&services).Error; err != nil {
		return nil, err
	}
	ops := make([]Operation, 0, len(services))
	for _, service := range services {
		ops = append(ops, r.serviceToOperation(service))
	}
	return ops, nil
}

func (r *Repository) GetOrderByPublished() (Order, error) {
	service, err := r.GetOnePublishedService()
	if err != nil || service == nil {
		if err != nil {
			return Order{}, err
		}
		fallback := fallbackOperations()[0]
		return Order{ID: fallback.ID, Operation: fallback, Result: "1000 мл"}, nil
	}
	return Order{ID: int(service.ID), Operation: r.serviceToOperation(*service), Result: "1000 мл"}, nil
}

func (r *Repository) SearchPublishedByName(name string) ([]Operation, error) {
	if r.DB == nil {
		return nil, nil
	}
	var services []db.Service
	pattern := "%" + strings.TrimSpace(name) + "%"
	if err := r.DB.Where("status = ? AND LOWER(name) LIKE LOWER(?)", db.StatusPublished, pattern).Order("date_created DESC").Find(&services).Error; err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, len(services))
	for _, service := range services {
		operations = append(operations, r.serviceToOperation(service))
	}
	return operations, nil
}

func (r *Repository) GetPublishedServiceByID(id int) (*db.Service, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}
	var service db.Service
	if err := r.DB.Where("id = ? AND status = ?", id, db.StatusPublished).First(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

func (r *Repository) SaveDraft(service *db.Service) error {
	if r.DB == nil {
		return errors.New("database is not available")
	}
	return r.DB.Save(service).Error
}

func (r *Repository) GetServiceInterfaceByID(id int) (*db.Service, error) {
	return r.FindServiceByID(id)
}

func (r *Repository) GetDraftOrPublishedByID(id int) (*db.Service, error) {
	if r.DB == nil {
		return nil, errors.New("database is not available")
	}
	var service db.Service
	if err := r.DB.First(&service, id).Error; err != nil {
		return nil, err
	}
	if service.Status == db.StatusDeleted {
		return nil, fmt.Errorf("service not found")
	}
	return &service, nil
}
