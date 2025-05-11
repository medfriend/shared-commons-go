package entity

import "time"

func (Migracion) TableName() string {
	return "migracion"
}

type Migracion struct {
	ID               uint      `gorm:"primaryKey;column:migracion_id" json:"migracion_id"`
	Version          string    `gorm:"type:varchar(255);not null;uniqueIndex;column:version" json:"version"`
	Nombre           string    `gorm:"type:varchar(255);not null;column:nombre" json:"nombre"`
	FechaAplicacion  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;column:fecha_aplicacion" json:"fecha_aplicacion"`
	SumaVerificacion string    `gorm:"type:varchar(64);not null;column:suma_verificacion" json:"suma_verificacion"`
	TiempoEjecucion  int       `gorm:"not null;column:tiempo_ejecucion" json:"tiempo_ejecucion"`
}
