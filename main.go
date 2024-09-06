package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-fastapi"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const netbootDir = "/var/www/netboot"

// https://github.com/swagger-api/swagger-ui
//
//go:embed swagger
var swagger embed.FS

// https://github.com/Redocly/redoc
//
//go:embed redoc
var redoc embed.FS

type Config struct {
	Address string `json:"address"`
	OS      string `json:"os"`
	Version string `json:"version"`
	Config  string `json:"config"`
}

type Host struct {
	Address string `json:"address"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type HostListResponse struct {
	Success   bool     `json:"success"`
	Message   string   `json:"message"`
	Addresses []string `json:"addresses"`
}

var MAC_PATTERN = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$`)

func AddHostHandler(ctx *gin.Context, in Config) (out Response, herr error) {

	out.Success = false
	if !MAC_PATTERN.MatchString(in.Address) {
		out.Message = "invalid MAC address"
		return
	}
	var osMenuPathname string
	var responsePathname string
	switch in.OS {
	case "debian":
		osMenuPathname = filepath.Join(netbootDir, "netboot-debian.ipxe")
		responsePathname = filepath.Join(netbootDir, fmt.Sprintf("%s-preseed.conf", in.Address))
	case "openbsd":
		osMenuPathname = filepath.Join(netbootDir, "netboot-openbsd.ipxe")
		responsePathname = filepath.Join(netbootDir, fmt.Sprintf("%s-install.conf", in.Address))
	default:
		out.Message = "Unrecognized OS"
		return
	}

	osFile, err := os.Open(osMenuPathname)
	if err != nil {
		out.Message = fmt.Sprintf("failed opening %s: %v", osMenuPathname, err)
		return
	}
	defer osFile.Close()
	hostMenuPathname := filepath.Join(netbootDir, fmt.Sprintf("%s.ipxe", in.Address))
	hostFile, err := os.Create(hostMenuPathname)
	if err != nil {
		out.Message = fmt.Sprintf("failed opening %s: %v", hostMenuPathname, err)
		return
	}
	defer hostFile.Close()
	_, err = io.Copy(osFile, hostFile)
	if err != nil {
		out.Message = fmt.Sprintf("failed copying %s to %s: %v", osMenuPathname, hostMenuPathname, err)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(in.Config)
	if err != nil {
		out.Message = fmt.Sprintf("failed decoding response data: %v", err)
		return
	}

	err = os.WriteFile(responsePathname, decodedBytes, 0640)
	if err != nil {
		out.Message = fmt.Sprintf("failed writing %s: %v", responsePathname, err)
		return
	}

	if in.OS == "debian" {
		err = compileInitrd(in, responsePathname)
		if err != nil {
			out.Message = fmt.Sprintf("failed compiling initrd: %v", err)
			return
		}
	}

	out.Success = true
	out.Message = fmt.Sprintf("%s configured", in.Address)
	return
}

func compileInitrd(in Config, responsePathname string) error {
	return fmt.Errorf("unimplemented")
}

func DeleteHostHandler(ctx *gin.Context, in Host) (out Response, err error) {
	out.Success = false
	out.Message = "DeleteHost unimplemented"
	return
}

func ListHostsHandler(ctx *gin.Context, in Host) (out HostListResponse, err error) {
	out.Success = false
	out.Message = "ListHosts unimplemented"
	out.Addresses = make([]string, 0)
	return
}

func CheckErr(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
		os.Exit(1)
	}
}

func main() {
	r := gin.Default()

	authorized := r.Group("/api", gin.BasicAuth(gin.Accounts{
		"admin": "admin",
	}))

	router := fastapi.NewRouter()
	router.AddCall("/add", AddHostHandler)
	router.AddCall("/delete", DeleteHostHandler)
	router.AddCall("/list", ListHostsHandler)

	err := AddOpenAPIDocs(r, router, "Reliance Systems Netboot Management API", true, true)
	CheckErr("Adding OpenAPI docs", err)

	authorized.POST("/api/*path", router.GinHandler)
	r.Run()
}

func AddOpenAPIDocs(engine *gin.Engine, router *fastapi.Router, title string, enableSwagger, enableRedoc bool) error {
	if enableSwagger {
		swaggerFiles, err := fs.Sub(swagger, "swagger")
		if err != nil {
			return fmt.Errorf("opening swagger subdirectory: %v", err)
		}
		engine.StaticFS("/swagger", http.FS(swaggerFiles))
	}

	if enableRedoc {
		redocFiles, err := fs.Sub(redoc, "redoc")
		if err != nil {
			return fmt.Errorf("opening redoc subdirectory: %v", err)
		}
		engine.StaticFS("/redoc", http.FS(redocFiles))

	}

	engine.GET("/v2/swagger.json", func(c *gin.Context) {
		swagger := router.EmitOpenAPIDefinition()
		swagger.Info.Title = title
		c.JSON(http.StatusOK, swagger)
	})
	return nil
}
