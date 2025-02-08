package files

import (
	"bufio"
	"fmt"
	"os"
)

func WriteToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo: %w", err)
	}

	if err = writer.Flush(); err != nil {
		return fmt.Errorf("error al vaciar el buffer: %w", err)
	}

	return nil
}
