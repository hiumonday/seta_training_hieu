package middleware

import (
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

func SetupPrometheus(r *gin.Engine) {
	p := ginprometheus.NewWithConfig(ginprometheus.Config{
		Subsystem: "gin",
	})

	p.Use(r)
}
