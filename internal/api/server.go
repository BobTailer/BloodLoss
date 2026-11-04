package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"BloodLoss/internal/app/handler"
	"BloodLoss/internal/app/repository"

	"github.com/gin-gonic/gin"
)

func projectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	if filepath.Base(cwd) == "api" && filepath.Base(filepath.Dir(cwd)) == "internal" {
		return filepath.Dir(filepath.Dir(cwd))
	}

	return cwd
}

func SetupRouter() *gin.Engine {
	repo := repository.NewRepository()
	h := handler.NewHandler(repo)
	r := gin.Default()

	root := projectRoot()

	// Подключаем шаблоны
	r.LoadHTMLGlob(filepath.Join(root, "templates", "*"))

	// Подключаем статику (картинки, css)
	r.Static("/static", filepath.Join(root, "internal", "resourses"))

	// Маршруты
	r.GET("/", h.ServicesList)
	r.GET("/services", h.ServicesList)
	r.GET("/add", h.AddServicePage)
	r.GET("/operation/:id", h.OperationDetail)
	r.GET("/order", func(ctx *gin.Context) {
		if service, err := repo.GetOnePublishedService(); err == nil && service != nil {
			ctx.Redirect(http.StatusFound, "/order/"+strconv.Itoa(int(service.ID)))
			return
		}
		ctx.Redirect(http.StatusFound, "/order/1")
	})
	r.GET("/order/:id", h.OrderDetail)
	r.POST("/select/:id", h.SelectOperation)
	r.POST("/services/create", h.CreateDraft)
	r.POST("/services/publish", h.PublishService)
	r.POST("/services/delete/:id", h.DeleteService)

	return r
}

func StartServer() {
	log.Println("Starting server")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	r := SetupRouter()
	log.Printf("Listening on :%s", port)
	r.Run(":" + port)
	log.Println("Server down")
}
