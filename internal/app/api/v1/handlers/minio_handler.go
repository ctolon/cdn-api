package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"

	"github.com/ctolon/cdn-api/internal/app/service"
	"github.com/ctolon/cdn-api/internal/utils"
)

type MinioHandler struct {
	minioService *service.MinioService
}

func NewMinioHandler(minioService *service.MinioService) *MinioHandler {
	return &MinioHandler{
		minioService: minioService,
	}
}
func (h *MinioHandler) ListObjects(c echo.Context) error {
	bucketName := c.QueryParam("bucketName")
	objectChan := h.minioService.ListObjects(c.Request().Context(), bucketName, minio.ListObjectsOptions{})

	var objects []minio.ObjectInfo
	for object := range objectChan {
		if object.Err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": object.Err.Error()})
		}
		objects = append(objects, object)
	}

	return c.JSON(http.StatusOK, objects)
}

func (h *MinioHandler) GetObject(c echo.Context) error {
	bucketName := c.QueryParam("bucketName")
	objectName := c.QueryParam("objectName")

	var width uint
	var height uint
	var resize bool

	if found, err := h.minioService.BucketExists(c.Request().Context(), bucketName); !found || err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	object, err := h.minioService.GetObject(c.Request().Context(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	// Convert Byte
	getByte := utils.StreamToByte(object)
	if len(getByte) == 0 {
		return c.File("./public/notfound.png")
	}

	// get size
	if err, orjWidth, orjHeight := utils.ImagickGetWidthHeight(getByte); err == nil {
		c.Set("Width", strconv.Itoa(int(orjWidth)))
		c.Set("Height", strconv.Itoa(int(orjHeight)))
	}

	// Set Content Type
	c.Response().Header().Set("Content-Type", http.DetectContentType(getByte))

	// Send Resized Image
	if resize {
		return c.Blob(http.StatusOK, http.DetectContentType(getByte), utils.ImagickResize(getByte, width, height))
	}

	// Send Original Image
	c.Response().WriteHeader(http.StatusFound)
	return c.Blob(http.StatusFound, http.DetectContentType(getByte), getByte)
}
