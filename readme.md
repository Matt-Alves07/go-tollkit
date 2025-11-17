# Toolbox

A simple example of how to create a reusable Go module with commonly used tools.

The included tools are:

- [X] Read JSON
- [X] Write JSON
- [X] Produce a JSON encoded error response
- [X] Upload a file to a specified directory
- [X] Upload multiple files to a specified directory
- [X] Download a static file
- [X] Get a random string of length n
- [X] Post JSON to a remote service
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
	http.HandleFunc("/write-json", writeJSONHandler)
	http.HandleFunc("/read-json", readJSONHandler)
	http.HandleFunc("/error-json", errorJSONHandler)
	http.HandleFunc("/post-json", postJSONHandler)

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

// readJSONHandler demonstra o uso da função ReadJSON.
func readJSONHandler(w http.ResponseWriter, r *http.Request) {
	var tools toolkit.Tools
	var payload struct {
		Foo string `json:"foo"`
	}

	err := tools.ReadJSON(w, r, &payload)
	if err != nil {
		_ = tools.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	response := struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{
		Error:   false,
		Message: fmt.Sprintf("Recebido o valor: %s", payload.Foo),
	}

	_ = tools.WriteJSON(w, http.StatusAccepted, response)
}

// writeJSONHandler demonstra o uso da função WriteJSON.
func writeJSONHandler(w http.ResponseWriter, r *http.Request) {
	var tools toolkit.Tools

	payload := struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{
		Error:   false,
		Message: "Esta é uma resposta JSON de exemplo",
	}

	_ = tools.WriteJSON(w, http.StatusOK, payload)
}

// errorJSONHandler demonstra o uso da função ErrorJSON.
func errorJSONHandler(w http.ResponseWriter, r *http.Request) {
	var tools toolkit.Tools

	testError := errors.New("este é um erro de teste")

	// Verifica se um status customizado foi passado via query param
	// Para testar, acesse: http://localhost:8080/error-json?status=custom
	status := r.URL.Query().Get("status")
	if status == "custom" {
		// Usa ErrorJSON com um status code customizado (401 Unauthorized)
		_ = tools.ErrorJSON(w, testError, http.StatusUnauthorized)
		return
	}

	// Por padrão, usa ErrorJSON com o status code padrão (400 Bad Request)
	// Para testar, acesse: http://localhost:8080/error-json
	_ = tools.ErrorJSON(w, testError)
}

// postJSONHandler demonstra o uso da função PushJSONToRemote.
// Este handler simula o envio de dados para um endpoint que, por sua vez, os recebe.
func postJSONHandler(w http.ResponseWriter, r *http.Request) {
	var tools toolkit.Tools

	// Dados a serem enviados
	payload := struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}{
		Name:  "exemplo",
		Value: 42,
	}

	// O endpoint de destino será o nosso próprio /read-json para demonstração.
	// Em um caso real, seria uma URL externa.
	// É necessário que o servidor esteja rodando para que o endpoint de destino exista.
	uri := "http://localhost:8080/read-json"

	jsonResponse, statusCode, err := tools.PushJSONToRemote(uri, payload)
	if err != nil {
		_ = tools.ErrorJSON(w, err)
		return
	}

	_ = tools.WriteJSON(w, statusCode, jsonResponse)
}
```
