package handler

import (
	"BloodLoss/internal/app/repository"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repo *repository.Repository
}

const defaultMediaURL = "http://localhost:9000/media"

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repo: r}
}

func mediaURL() string {
	if value := os.Getenv("MINIO_MEDIA_URL"); value != "" {
		return value
	}
	return defaultMediaURL
}

func (h *Handler) ServicesList(ctx *gin.Context) {
	query := ctx.Query("search")
	minHbStr := ctx.Query("min_hb")
	operations := h.Repo.FilterOperations(query)

	if minHbStr != "" {
		if minHb, err := strconv.ParseFloat(minHbStr, 64); err == nil {
			filtered := make([]repository.Operation, 0)
			for _, op := range operations {
				if op.HbBefore >= minHb {
					filtered = append(filtered, op)
				}
			}
			operations = filtered
		}
	}

	ctx.HTML(http.StatusOK, "services.html", gin.H{
		"operations": operations,
		"search":     query,
		"minHb":      minHbStr,
		"mediaURL":   mediaURL(),
	})
}

func (h *Handler) OperationDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.Atoi(idStr)
	operation, ok := h.Repo.GetOperationByID(id)
	if !ok {
		operation = repository.Operation{Name: "Новая услуга", Description: "Новая запись", Avatar: "cesarean.svg", Photo: "cesarean.svg", Video: "cesarean-animated.svg"}
	}
	ctx.HTML(http.StatusOK, "operation.html", gin.H{
		"operation": operation,
		"mediaURL":  mediaURL(),
		"mode":      "publish",
	})
}

func (h *Handler) OrderDetail(ctx *gin.Context) {
	operations, err := h.Repo.GetPublishedServices()
	if err != nil {
		operations = h.Repo.GetOperations()
	}
	ctx.HTML(http.StatusOK, "order.html", gin.H{
		"operations": operations,
		"mediaURL":   mediaURL(),
	})
}

func (h *Handler) AddServicePage(ctx *gin.Context) {
	draft, err := h.Repo.GetFirstDraftByUser(1)
	if err != nil {
		draft = nil
	}
	ctx.HTML(http.StatusOK, "add.html", gin.H{
		"draft":    draft,
		"mediaURL": mediaURL(),
	})
}

func (h *Handler) CreateDraft(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.PostForm("name"))
	description := strings.TrimSpace(ctx.PostForm("description"))
	imageURL := strings.TrimSpace(ctx.PostForm("imageURL"))
	videoURL := strings.TrimSpace(ctx.PostForm("videoURL"))
	beforeVal, _ := strconv.ParseFloat(strings.TrimSpace(ctx.PostForm("hb_before")), 64)
	afterVal, _ := strconv.ParseFloat(strings.TrimSpace(ctx.PostForm("hb_after")), 64)
	if name == "" {
		name = "Новая услуга"
	}
	if description == "" {
		description = "Описание услуги"
	}
	if imageURL == "" {
		imageURL = "cesarean.svg"
	}
	if videoURL == "" {
		videoURL = "cesarean-animated.svg"
	}

	service, err := h.Repo.CreateDraft(1, name, description, imageURL, videoURL, beforeVal, afterVal)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка создания черновика: %v", err)
		return
	}
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/operation/%d", service.ID))
}

func (h *Handler) PublishService(ctx *gin.Context) {
	idStr := strings.TrimSpace(ctx.PostForm("id"))
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ctx.String(http.StatusBadRequest, "Некорректный id услуги")
		return
	}
	name := strings.TrimSpace(ctx.PostForm("name"))
	description := strings.TrimSpace(ctx.PostForm("description"))
	if name == "" {
		name = "Новая услуга"
	}
	if description == "" {
		description = "Описание услуги"
	}
	beforeVal, _ := strconv.ParseFloat(strings.TrimSpace(ctx.PostForm("hb_before")), 64)
	afterVal, _ := strconv.ParseFloat(strings.TrimSpace(ctx.PostForm("hb_after")), 64)

	service, err := h.Repo.GetServiceInterfaceByID(id)
	if err != nil || service == nil {
		ctx.String(http.StatusNotFound, "Услуга не найдена")
		return
	}
	service.Name = name
	service.ShortDescription = description
	service.HbBefore = beforeVal
	service.HbAfter = afterVal
	if err := h.Repo.SaveDraft(service); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка сохранения данных: %v", err)
		return
	}
	if err := h.Repo.PublishService(id, description, beforeVal, afterVal); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка публикации услуги: %v", err)
		return
	}
	ctx.Redirect(http.StatusFound, "/services")
}

func (h *Handler) DeleteService(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ctx.String(http.StatusBadRequest, "Некорректный id услуги")
		return
	}
	if err := h.Repo.DeleteService(id); err != nil {
		ctx.String(http.StatusInternalServerError, "Ошибка удаления: %v", err)
		return
	}
	ctx.Redirect(http.StatusFound, "/services")
}

func (h *Handler) SelectOperation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.Atoi(idStr)
	h.Repo.SelectOperation(id)
	ctx.Redirect(http.StatusFound, "/order/"+idStr)
}
