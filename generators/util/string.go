package util

import "unicode"

func CapitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	// Convierte el string en un slice de runas para manejar correctamente caracteres Unicode.
	runeStr := []rune(s)
	// Convierte la primera runa a mayúscula.
	runeStr[0] = unicode.ToUpper(runeStr[0])
	// Devuelve el nuevo string.
	return string(runeStr)
}
