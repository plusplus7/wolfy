package server

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
	"wolfy/service/bilibili"
)

type RemoteSignatoryServer struct {
	router    *gin.Engine
	signatory bilibili.ISignatory
}

func NewRemoteSignatory(accessKeyID, accessKeySecret string) *RemoteSignatoryServer {
	return &RemoteSignatoryServer{
		signatory: bilibili.NewLocalSignatory(accessKeyID, accessKeySecret),
		router:    gin.Default(),
	}
}

func (r *RemoteSignatoryServer) Spin() {
	err := r.router.Run("[::]:41376")

	if err != nil {
		panic(err)
	}
}

func (r *RemoteSignatoryServer) Register() {
	r.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			fmt.Println(origin)
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	r.router.POST("/sign", r.Sign)
}

func (r *RemoteSignatoryServer) Sign(c *gin.Context) {
	var req bilibili.RemoteSignRequest
	err := c.BindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	log.Println(req.AnchorCode, req.ReqJson)

	sign, err := r.signatory.Sign(req.ReqJson)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"signed": sign})
}
