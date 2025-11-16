package toolkit

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("Wrong length random string returned")
	}
}

func TestTools_UploadFile(t *testing.T) {
	// Configuração do teste
	var tools Tools
	tools.MaxFileSize = 1024 * 1024 * 1 // 1MB

	// Test cases
	testCases := []struct {
		name          string
		fileContent   string
		fileSize      int64
		renameFile    bool
		expectedError bool
		errorMsg      string
	}{
		{
			name:          "upload de arquivo válido sem renomear",
			fileContent:   "conteúdo do arquivo de teste",
			fileSize:      int64(len("conteúdo do arquivo de teste")),
			renameFile:    false,
			expectedError: false,
		},
		{
			name:          "upload de arquivo válido com renomeação",
			fileContent:   "outro conteúdo de arquivo",
			fileSize:      int64(len("outro conteúdo de arquivo")),
			renameFile:    true,
			expectedError: false,
		},
		{
			name:          "upload de arquivo muito grande",
			fileContent:   strings.Repeat("a", int(tools.MaxFileSize+1)),
			fileSize:      int64(tools.MaxFileSize) + 1,
			renameFile:    false,
			expectedError: true,
			errorMsg:      "request body too large",
		},
		{
			name:          "nenhum arquivo para upload",
			fileContent:   "",
			fileSize:      0,
			renameFile:    false,
			expectedError: true,
			errorMsg:      "no file uploaded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Cria um diretório temporário para uploads
			uploadDir, err := os.MkdirTemp("", "test_uploads")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(uploadDir) // Limpa o diretório após o teste

			var req *http.Request

			if tc.name == "upload de arquivo muito grande" {
				// Para simular um erro de corpo de requisição muito grande, precisamos de um servidor real
				// que aplique o limite com http.MaxBytesReader.
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Envolve o corpo da requisição com MaxBytesReader para impor o limite de tamanho.
					r.Body = http.MaxBytesReader(w, r.Body, int64(tools.MaxFileSize))
					_, err := tools.UploadFile(r, uploadDir, tc.renameFile)
					if err == nil {
						t.Error("esperado um erro de arquivo muito grande, mas não ocorreu")
						w.WriteHeader(http.StatusOK)
						return
					}
					if !strings.Contains(err.Error(), "request body too large") {
						t.Errorf("esperado erro 'request body too large', mas ocorreu '%s'", err.Error())
					}
					w.WriteHeader(http.StatusRequestEntityTooLarge)
				})

				server := httptest.NewServer(handler)
				defer server.Close()

				// Cria o corpo da requisição
				body := new(bytes.Buffer)
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("file", "testfile.txt")
				_, _ = io.Copy(part, strings.NewReader(tc.fileContent))
				writer.Close()

				// Envia a requisição para o servidor de teste
				resp, err := http.Post(server.URL, writer.FormDataContentType(), body)
				if err != nil {
					t.Fatalf("Erro ao postar para o servidor de teste: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusRequestEntityTooLarge {
					t.Errorf("Esperado status %d, mas obteve %d", http.StatusRequestEntityTooLarge, resp.StatusCode)
				}
				return // O teste para este caso termina aqui.
			}

			// Para outros casos, continue com o httptest.NewRequest
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			if tc.fileContent != "" {
				part, _ := writer.CreateFormFile("file", "testfile.txt")
				_, _ = io.Copy(part, strings.NewReader(tc.fileContent))
			}
			writer.Close()

			req = httptest.NewRequest("POST", "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Chama a função a ser testada
			uploadedFile, err := tools.UploadFile(req, uploadDir, tc.renameFile)

			// Verifica se o erro é o esperado
			if tc.expectedError {
				if err == nil {
					t.Fatalf("esperado um erro, mas não ocorreu")
				}
				if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("esperado erro '%s', mas ocorreu '%s'", tc.errorMsg, err.Error())
				}
				return // Fim do teste para este caso
			}

			// Se não houver erro esperado, verifica o sucesso do upload
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}

			if uploadedFile == nil {
				t.Fatal("uploadedFile não deve ser nulo")
			}

			// Verifica o nome do arquivo original
			if uploadedFile.OriginalFileName != "testfile.txt" {
				t.Errorf("nome do arquivo original incorreto: esperado 'testfile.txt', obteve '%s'", uploadedFile.OriginalFileName)
			}

			// Verifica o tamanho do arquivo
			if int64(uploadedFile.FileSize) != tc.fileSize {
				t.Errorf("tamanho do arquivo incorreto: esperado %d, obteve %d", tc.fileSize, int64(uploadedFile.FileSize))
			}

			// Verifica se o arquivo foi renomeado corretamente
			if tc.renameFile {
				if uploadedFile.NewFileName == uploadedFile.OriginalFileName {
					t.Error("o arquivo deveria ter sido renomeado, mas não foi")
				}
			} else {
				if uploadedFile.NewFileName != uploadedFile.OriginalFileName {
					t.Error("o arquivo não deveria ter sido renomeado, mas foi")
				}
			}

			// Verifica se o arquivo existe no diretório de upload
			filePath := filepath.Join(uploadDir, uploadedFile.NewFileName)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("o arquivo enviado não foi encontrado no diretório de upload: %s", filePath)
			}
		})
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool // Corrigido para errorExpected
	errorMsg      string
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true, errorMsg: "file type image/png not allowed"},
}

func TestTools_UploadFiles(t *testing.T) {
	// Cria o diretório de upload de teste se ele não existir
	uploadPath := "./testdata/uploads"
	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		err := os.MkdirAll(uploadPath, os.ModePerm)
		if err != nil {
			t.Fatalf("Não foi possível criar o diretório de teste: %v", err)
		}
	}

	for _, e := range uploadTests {
		t.Run(e.name, func(t *testing.T) {
			// Cria um buffer para o corpo da requisição multipart
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)

			// Cria uma imagem PNG de 1x1 pixel para o teste
			part, err := writer.CreateFormFile("file", "test.png")
			if err != nil {
				t.Fatalf("CreateFormFile failed: %v", err)
			}

			img := image.NewRGBA(image.Rect(0, 0, 1, 1))
			err = png.Encode(part, img)
			if err != nil {
				t.Fatalf("png.Encode failed: %v", err)
			}
			writer.Close()

			// Cria a requisição HTTP
			request := httptest.NewRequest("POST", "/", body)
			request.Header.Add("Content-Type", writer.FormDataContentType())

			var testTools Tools
			testTools.AllowedTypes = e.allowedTypes

			uploadedFiles, err := testTools.UploadFiles(request, uploadPath, e.renameFile)
			if err != nil && !e.errorExpected {
				t.Errorf("%s: erro inesperado: %v", e.name, err)
			}

			if !e.errorExpected {
				if len(uploadedFiles) == 0 {
					t.Fatalf("%s: nenhum arquivo foi enviado", e.name)
				}
				if _, err := os.Stat(filepath.Join(uploadPath, uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
					t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
				}
				// clean up
				_ = os.Remove(filepath.Join(uploadPath, uploadedFiles[0].NewFileName))
			} else { // Error was expected
				if err == nil {
					t.Errorf("%s: um erro era esperado, mas nenhum foi recebido", e.name)
				} else if e.errorMsg != "" && !strings.Contains(err.Error(), e.errorMsg) {
					t.Errorf("%s: esperava erro '%s', mas obteve '%s'", e.name, e.errorMsg, err.Error())
				}
			}
		})
	}

	// Limpa o diretório de uploads após todos os testes
	_ = os.RemoveAll(uploadPath)
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools

	t.Run("cria diretório que não existe", func(t *testing.T) {
		// Define um caminho de diretório temporário que certamente não existe
		tempDir := filepath.Join(os.TempDir(), "test_create_dir")
		// Garante que o diretório seja removido no final do teste
		defer os.RemoveAll(tempDir)

		// Remove o diretório para garantir que o teste comece do zero
		_ = os.RemoveAll(tempDir)

		err := testTools.CreateDirIfNotExist(tempDir)
		if err != nil {
			t.Fatalf("CreateDirIfNotExist retornou um erro inesperado: %v", err)
		}

		// Verifica se o diretório foi realmente criado
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Error("O diretório não foi criado, mas deveria ter sido")
		}
	})

	t.Run("tenta criar diretório que já existe", func(t *testing.T) {
		// Cria um diretório temporário que já existe
		tempDir, err := os.MkdirTemp("", "test_dir_exists")
		if err != nil {
			t.Fatalf("Falha ao criar diretório temporário: %v", err)
		}
		defer os.RemoveAll(tempDir)

		err = testTools.CreateDirIfNotExist(tempDir)
		if err != nil {
			t.Errorf("CreateDirIfNotExist retornou um erro para um diretório existente: %v", err)
		}
	})
}
