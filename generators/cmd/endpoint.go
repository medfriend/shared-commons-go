/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/files"
	"github.com/medfriend/shared-commons-go/generators/template"

	"github.com/spf13/cobra"
)

// endpointCmd represents the endpoint command
var endpointCmd = &cobra.Command{
	Use:   "endpoint [name] [package]",
	Short: "Creacion de un endpoint",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("endpoint called")

		files.WriteToFile(fmt.Sprintf("entity/%s.entity.go", args[0]), template.GetEntity(args))
		files.WriteToFile(fmt.Sprintf("repository/%s.repository.go", args[0]), template.GetRepositoryTemplate(args))
		files.WriteToFile(fmt.Sprintf("service/%s.service.go", args[0]), template.GetService(args))
		files.WriteToFile(fmt.Sprintf("controller/%s.controller.go", args[0]), template.GetController(args))
		files.WriteToFile(fmt.Sprintf("module/%s.module.go", args[0]), template.GetModule(args))
		files.WriteToFile(fmt.Sprintf("router/%s.router.go", args[0]), template.GetRoute(args))

		// Ejecutar swag init
		if err := executeCommand("swag", "init"); err != nil {
			fmt.Println("Error ejecutando swag init:", err)
			return
		}

		if err := executeCommandInDir("module", "wire"); err != nil {
			fmt.Println("Error wire", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(endpointCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// endpointCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// endpointCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
