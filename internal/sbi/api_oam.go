package sbi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/nwdaf/pkg/components"
)

func (s *Server) getOamRoutes() []Route {
	return []Route{
		{
			Name:    "Health Check",
			Method:  http.MethodGet,
			Pattern: "/",
			APIFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "CHF OAM woking!")
			},
		},
		{
			Name:    "NfResourceGet",
			Method:  http.MethodGet,
			Pattern: "/nf-resource",
			APIFunc: s.ChfOamNfResourceGet,
		},
	}
}

func (s *Server) ChfOamNfResourceGet(c *gin.Context) {
	nfResource, err := components.GetNfResouces(context.Background())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	fmt.Println(nfResource)
	c.JSON(http.StatusOK, *nfResource)
}
