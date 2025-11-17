# Toolbox

A simple example of how to create a reusable Go module with commonly used tools.

The included tools are:

- [X] Read JSON
- [ ] Write JSON
- [ ] Produce a JSON encoded error response
- [X] Upload a file to a specified directory
- [X] Upload multiple files to a specified directory
- [X] Download a static file
- [X] Get a random string of length n
- [ ] Post JSON to a remote service 
- [X] Create a directory, including all parent directories, if it does not already exist
- [X] Create a URL safe slug from a string

## Installation

`go get -u github.com/Matt-Alves07/go-toolkit`

## Usage

`import "github.com/Matt-Alves07/go-toolkit/toolkit"`

## Examples

```go 
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Matt-Alves07/go-toolkit"
)

func main() {
	// --- Exemplos Básicos ---
	runBasicExamples()

	// --- Exemplos Web ---
	// Configura os handlers para as rotas de upload
	http.HandleFunc("/upload-single", uploadSingleFileHandler)
	http.HandleFunc("/upload-multiple", uploadMultipleFilesHandler)

	fmt.Println("Servidor iniciado na porta 8080")
	fmt.Println("Use os endpoints /upload-single ou /upload-multiple para testar.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func runBasicExamples() {
	var tools toolkit.Tools

	// Exemplo de RandomString
	randomStr := tools.RandomString(10)
	fmt.Println("Exemplo de RandomString:", randomStr)
	// Resultado esperado: uma string aleatória com 10 caracteres, ex: "aB1cD2eF3g"

	// Exemplo de Slugify
	slug, err := tools.Slugify("Olá, Mundo! 123?")
	if err != nil {
		log.Println(err)
	}
	fmt.Println("Exemplo de Slug:", slug)
	// Resultado esperado: "ol-mundo-123"

	// Exemplo de CreateDirIfNotExist
	err = tools.CreateDirIfNotExist("./meu-diretorio-de-teste")
	if err != nil {
		log.Println(err)
	}
	fmt.Println("Diretório 'meu-diretorio-de-teste' criado (se não existia).")
	// Resultado esperado: um diretório chamado 'meu-diretorio-de-teste' será criado na raiz do projeto.
}

// uploadSingleFileHandler lida com o upload de um único arquivo.
func uploadSingleFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var tools toolkit.Tools
	tools.AllowedTypes = []string{"image/jpeg", "image/png", "text/plain"}

	uploadedFile, err := tools.UploadFile(r, "./uploads", true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Arquivo enviado com sucesso! Nome original: %s, Novo nome: %s, Tamanho: %d bytes",
		uploadedFile.OriginalFileName, uploadedFile.NewFileName, uploadedFile.FileSize)
}

// uploadMultipleFilesHandler lida com o upload de múltiplos arquivos.
func uploadMultipleFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var tools toolkit.Tools
	tools.AllowedTypes = []string{"image/jpeg", "image/png", "text/plain"}

	uploadedFiles, err := tools.UploadFiles(r, "./uploads", true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Foram enviados %d arquivos.\n", len(uploadedFiles))
	for _, file := range uploadedFiles {
		fmt.Fprintf(w, "- Nome original: %s, Novo nome: %s, Tamanho: %d bytes\n",
			file.OriginalFileName, file.NewFileName, file.FileSize)
	}
}
```
