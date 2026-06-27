package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
	"wolfy/components/danmu/bilibili"
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
		log.Println(err)
	}
}

func (r *RemoteSignatoryServer) Register() {
	r.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001"},
		AllowMethods:     []string{"POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
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
