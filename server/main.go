package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"sync"
)

var hubs = sync.Map{}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	r := gin.Default()
	r.GET("/chat", serve)
	r.Run()
}
