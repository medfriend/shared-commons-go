package minio

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/consul"
	"github.com/medfriend/shared-commons-go/util/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func CONN(consulClient *api.Client) *minio.Client {
	minioInfo, _ := consul.GetKeyValue(consulClient, "MINIO")

	var minioInfoResult map[string]string
	err := json.Unmarshal([]byte(minioInfo), &minioInfoResult)

	if err != nil {
	}

	endpoint := minioInfoResult["MINIO_HOST"] + ":" + minioInfoResult["MINIO_PORT"]
	accessKeyID := minioInfoResult["MINIO_ACCESS"]
	secretAccessKey := minioInfoResult["MINIO_SECRET"]
	useSSL := false

	// Solo el destino: la ficha trae MINIO_ACCESS y MINIO_SECRET.
	logger.GetLogger().Debug("Información MinIO obtenida", zap.String("endpoint", endpoint))

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.GetLogger().Fatal("Error al crear cliente MinIO", zap.Error(err))
	}

	logger.GetLogger().Info("Conectado a MinIO", zap.String("endpoint", endpoint))

	// Listar todos los buckets
	ctx := context.Background()
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		logger.GetLogger().Fatal("Error al listar buckets", zap.Error(err))
	}

	for _, bucket := range buckets {
		logger.GetLogger().Info("Bucket encontrado", zap.String("bucket", bucket.Name))
	}

	return minioClient
}
