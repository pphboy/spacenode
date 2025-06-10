package spacehttp

import (
	"fmt"
	"spacenode/libs/lzcutils"
	"spacenode/libs/models"
	"spacenode/modules/appaider"
	"spacenode/modules/db"
	"spacenode/modules/lzcapp"
	"spacenode/modules/space"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Server struct {
	engin        *gin.Engine
	port         int
	spaceManager *space.Space
	appAider     appaider.AppAider
	lzcapp       lzcapp.LzcAppManager
}

func NewServer(port int) (*Server, error) {
	lzcm, err := lzcapp.NewLzcAppManager()
	if err != nil {
		logrus.Errorf("failed to init lzcapp manager: %v", err)
		return nil, err
	}
	aa, err := appaider.NewAppAider(db.DB(), lzcm)
	if err != nil {
		return nil, err
	}

	// WARN: 目前是写死的，因为没有必要的过早进行 扩展式的设计
	sm, err := space.NewSpace(models.SpaceItemConfig{
		Port:    59393,
		Host:    "host.lzcapp",
		ID:      "space1",
		NetAddr: "172.168.1.0",
		Mask:    "255.255.255.0",
	})
	if err != nil {
		return nil, err
	}
	go func() {
		defer sm.Stop()
		if err := sm.Start(); err != nil {
			logrus.Errorln("space manager start failed: ", err)
		}
	}()

	s := &Server{
		engin:        gin.Default(),
		port:         port,
		lzcapp:       lzcm,
		appAider:     aa,
		spaceManager: sm,
	}
	s.register()
	return s, nil
}

func (s *Server) Start() error {
	return s.engin.Run(fmt.Sprintf(":%d", s.port))
}

func (s *Server) register() {
	s.registerSpaceManager(s.engin.Group("space"))
	s.registerAppAider(s.engin.Group("app"))
	s.registerLzcApp(s.engin.Group("lzcapp"))
}

func (s *Server) registerLzcApp(group *gin.RouterGroup) {

	group.GET("/applist", func(ctx *gin.Context) {
		apps, err := s.lzcapp.AppList(lzcutils.ToGrpcCtxFromGinCtx(ctx))
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(200, gin.H{"apps": apps})
	})
}

func (s *Server) registerSpaceManager(group *gin.RouterGroup) {
	group.GET("/list", func(ctx *gin.Context) {
		ctx.JSON(200, s.spaceManager.Nodelist())
	})
}

func (s *Server) registerAppAider(group *gin.RouterGroup) {
	group.GET("/list", func(ctx *gin.Context) {
		ctx.JSON(200, s.appAider.List())
	})

	group.POST("/add", func(ctx *gin.Context) {
		appid := ctx.Query("appid")
		if appid == "" {
			ctx.JSON(400, gin.H{"error": "appid is required"})
			return
		}

		if err := s.appAider.Add(lzcutils.ToGrpcCtxFromGinCtx(ctx), &models.AppNode{
			AppID:   appid,
			NodeID:  "lzcapp" + uuid.NewString()[6:],
			SpaceID: "space1",
		}); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			logrus.Errorf("add app error: %v", err)
			return
		}
		ctx.JSON(200, gin.H{"message": "add app success"})
	})

	group.POST("/remove", func(ctx *gin.Context) {
		appid := ctx.Query("appid")
		nodeid := ctx.Query("nodeid")
		if appid == "" {
			ctx.JSON(400, gin.H{"error": "appid is required"})
			return
		}
		if err := s.spaceManager.Remove(models.SpaceNode{
			NodeID: nodeid,
		}); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			logrus.Errorf("remove app error: %v", err)
			return
		}
		if err := s.appAider.Remove(lzcutils.ToGrpcCtxFromGinCtx(ctx), &models.AppNode{
			AppID:   appid,
			SpaceID: "space1",
		}); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			logrus.Errorf("remove app error: %v", err)
			return
		}

		ctx.JSON(200, gin.H{"message": "remove app success"})
	})
}
