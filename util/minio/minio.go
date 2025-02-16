package minio

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/consul"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
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

	fmt.Println(minioInfoResult)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Conectado a MinIO en %s\n", endpoint)

	// Listar todos los buckets
	ctx := context.Background()
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	for _, bucket := range buckets {
		log.Printf("Bucket encontrado: %s\n", bucket.Name)
	}

	return minioClient
}
