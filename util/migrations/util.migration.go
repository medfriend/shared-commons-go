package migrations

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/medfriend/shared-commons-go/util/entity"
	"github.com/medfriend/shared-commons-go/util/logger"
	"github.com/medfriend/shared-commons-go/util/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

func calcularChecksum(contenido string) string {
	hash := md5.Sum([]byte(contenido))
	return hex.EncodeToString(hash[:])
}

func ReadMigration(db *gorm.DB) error {
	// Verificar que db no sea nil
	if db == nil {
		return fmt.Errorf("la conexión a la base de datos es nula")
	}

	archivos, err := ioutil.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("error al leer el directorio de migraciones: %v", err)
	}

	// Inicializar el repositorio de migraciones
	migrationRepo := repository.NewMigrationRepository(db)

	for _, archivo := range archivos {
		if archivo.IsDir() || !strings.HasSuffix(archivo.Name(), ".sql") {
			continue
		}

		nombreArchivo := archivo.Name()

		// Verificar si la migración ya existe en la tabla de migraciones
		migration, err := migrationRepo.ExistByNombre(nombreArchivo)
		if err != nil {
			return fmt.Errorf("error al buscar la migración %s: %v", nombreArchivo, err)
		}

		// Si la migración ya existe, saltamos a la siguiente
		if migration != nil {
			logger.GetLogger().Info("Migración ya aplicada, saltando", zap.String("archivo", nombreArchivo))
			continue
		}

		// Leer el contenido del archivo
		rutaArchivo := filepath.Join("migrations", nombreArchivo)
		contenido, err := ioutil.ReadFile(rutaArchivo)
		if err != nil {
			return fmt.Errorf("error al leer el archivo %s: %v", nombreArchivo, err)
		}

		contenidoStr := string(contenido)

		// Ejecutar la migración con manejo específico de errores
		// Dividir el script SQL en sentencias individuales
		sentencias := splitSQL(contenidoStr)

		// Iniciar transacción
		tx := db.Begin()
		inicio := time.Now()

		logger.GetLogger().Info("Ejecutando migración", zap.String("archivo", nombreArchivo))

		erroresIgnorados := false

		// Ejecutar cada sentencia individualmente
		for _, sentencia := range sentencias {
			sentencia = strings.TrimSpace(sentencia)
			if sentencia == "" {
				continue
			}

			result := tx.Exec(sentencia)

			if result.Error != nil {
				// Verificar si es un error que podemos ignorar
				errorStr := result.Error.Error()
				if strings.Contains(errorStr, "already exists") ||
					strings.Contains(errorStr, "ya existe") {
					// Ignorar este error y continuar
					logger.GetLogger().Warn("Ignorando error en sentencia",
						zap.String("sentencia", sentencia[:Min(50, len(sentencia))]+"..."),
						zap.String("error", errorStr))
					erroresIgnorados = true
					continue
				} else {
					// Otro tipo de error, hacemos rollback
					tx.Rollback()
					return fmt.Errorf("error al ejecutar sentencia SQL: %v\nSentencia: %s",
						result.Error, sentencia[:Min(100, len(sentencia))]+"...")
				}
			}
		}

		duracion := time.Since(inicio).Milliseconds()

		// Crear registro de la migración
		nuevaMigracion := &entity.Migracion{
			Version:          ExtractVersion(nombreArchivo),
			Nombre:           nombreArchivo,
			FechaAplicacion:  time.Now(),
			SumaVerificacion: calcularChecksum(contenidoStr),
			TiempoEjecucion:  int(duracion),
		}

		// Guardar el registro de la migración
		if err := migrationRepo.Save(nuevaMigracion); err != nil {
			tx.Rollback()
			return fmt.Errorf("error al guardar registro de migración %s: %v", nombreArchivo, err)
		}

		tx.Commit()

		if erroresIgnorados {
			logger.GetLogger().Warn("Migración aplicada con advertencias",
				zap.String("archivo", nombreArchivo),
				zap.Int64("duracion_ms", duracion))
		} else {
			logger.GetLogger().Info("Migración aplicada correctamente",
				zap.String("archivo", nombreArchivo),
				zap.Int64("duracion_ms", duracion))
		}
	}

	return nil
}

// Función para dividir el script SQL en sentencias individuales
func splitSQL(script string) []string {
	// Dividir por punto y coma, pero teniendo cuidado con las comillas y comentarios
	// Esta es una implementación básica, se podría mejorar para manejar casos más complejos
	var sentencias []string
	var currentStmt string

	lines := strings.Split(script, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Saltar líneas vacías o comentarios completos
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// Eliminar comentarios al final de la línea
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}

		currentStmt += line + " "

		// Si la línea termina con punto y coma, es el final de una sentencia
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			sentencias = append(sentencias, strings.TrimSpace(currentStmt))
			currentStmt = ""
		}
	}

	// Agregar la última sentencia si existe
	if strings.TrimSpace(currentStmt) != "" {
		sentencias = append(sentencias, strings.TrimSpace(currentStmt))
	}

	return sentencias
}

// Función auxiliar para extraer la versión del nombre de archivo
func ExtractVersion(filename string) string {
	// Asumiendo que el formato es como "001_nombre.sql"
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "0"
}

// Función auxiliar Min para obtener el mínimo de dos números
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
