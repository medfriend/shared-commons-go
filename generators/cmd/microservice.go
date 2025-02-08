/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/files"
	"github.com/medfriend/shared-commons-go/generators/template"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

// microserviceCmd represents the microservice command
var microserviceCmd = &cobra.Command{
	Use:   "microservice [name] [port]",
	Short: "creacion de un microservicio completo",
	Run: func(cmd *cobra.Command, args []string) {

		dirName := fmt.Sprintf("%s-go", args[0])

		// Array de subdirectorios a crear dentro de dirName
		subDirs := []string{"httpServer", "controller", "entity", "module", "repository", "router", "service"}

		// Crear el directorio principal primero
		if err := os.Mkdir(dirName, 0755); err != nil {
			fmt.Println("Error al crear el directorio principal:", err)
			return
		}

		// Iterar sobre el array de subdirectorios y crear cada uno
		for _, subDir := range subDirs {
			fullPath := dirName + "/" + subDir
			if err := os.Mkdir(fullPath, 0755); err != nil {
				fmt.Println("Error al crear el subdirectorio:", err)
				return
			}
		}

		fmt.Println("Todos los directorios han sido creados exitosamente.")

		files.WriteToFile(dirName+"/main.go", template.GetMain(args))
		files.WriteToFile(dirName+"/.env", template.GetEnv(args))
		files.WriteToFile(dirName+"/httpServer/httpServer.go", template.GetHttpServerTemplate(args))
		files.WriteToFile(dirName+"/router/admin.router.go", template.GetAdminRouter(args))

		pwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error al obtener el directorio actual:", err)
			return
		}

		// Añadir el subdirectorio al path actual
		fullPath := filepath.Join(pwd, dirName)

		// Cambiar al nuevo directorio
		if err := os.Chdir(fullPath); err != nil {
			fmt.Println("Error al cambiar al directorio:", err)
			return
		}

		// Ejecutar go mod init
		if err := executeCommand("go", "mod", "init", dirName); err != nil {
			fmt.Println("Error ejecutando go mod init:", err)
			return
		}

		if err := executeCommand("go", "mod", "tidy"); err != nil {
			fmt.Println("Error ejecutando go mod tidy:", err)
			return
		}

		// Ejecutar swag init
		if err := executeCommand("swag", "init"); err != nil {
			fmt.Println("Error ejecutando swag init:", err)
			return
		}
	},
}

func executeCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(microserviceCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// microserviceCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// microserviceCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
