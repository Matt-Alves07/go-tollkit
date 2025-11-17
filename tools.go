package toolkit

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is a utility struct that provides various helper methods.
type Tools struct{
	MaxFileSize			int
	AllowedTypes		[]string
	MaxJSONSize			int
	AllowUnknownFields	bool
}

// RandomString generates a random string of the specified length n.
// The string consists of uppercase and lowercase letters, digits, and the characters '_' and '+'.
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)

	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))

		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile represents an uploaded file with its new name, original name, and size.
type UploadedFile struct {
	NewFileName string
	OriginalFileName string
	FileSize uint64
}

func (t *Tools) UploadFiles(r * http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFiles := true
	if len(rename) > 0 {
		renameFiles = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 10 // 10 MB default
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, err
	}

	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			uploadedFile, err := t.processUploadedFile(hdr, uploadDir, renameFiles)
			if err != nil { // This now correctly handles file type errors
				return nil, err
			}
			uploadedFiles = append(uploadedFiles, uploadedFile)
		}
	}

	    return uploadedFiles, nil
}
	
// UploadFile sobe um único arquivo para o servidor. Se múltiplos arquivos forem enviados no request,
// apenas o primeiro será processado.
func (t *Tools) UploadFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFile *UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 10 // 10 MB default
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, err
	}

	if files, ok := r.MultipartForm.File["file"]; ok {
		if len(files) > 0 {
			hdr := files[0]
			uploadedFile, err = t.processUploadedFile(hdr, uploadDir, renameFile)
			if err != nil {
				return uploadedFile, err
			}
		}
	}

	if uploadedFile == nil {

		return nil, errors.New("no file uploaded")
	}

	return uploadedFile, nil
}

func (t *Tools) processUploadedFile(hdr *multipart.FileHeader, uploadDir string, renameFile bool) (*UploadedFile, error) {
	var uploadedFile UploadedFile
	infile, err := hdr.Open()
	if err != nil {
		return nil, err	}
	defer infile.Close()

	// Checa o tipo do arquivo
	if len(t.AllowedTypes) > 0 {
		fileBytes, err := io.ReadAll(infile)
		if err != nil {
			return nil, err
		}
		// Volta o ponteiro do arquivo para o início
		_, err = infile.Seek(0, 0)
		if err != nil {
			return nil, err
		}

		fileType := http.DetectContentType(fileBytes)
		if !t.isAllowedType(fileType) {
			return nil, fmt.Errorf("file type %s not allowed", fileType)
		}
	}

	uploadedFile.OriginalFileName = hdr.Filename

	var outfile *os.File
	if renameFile {
		uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(hdr.Filename))
		outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName))
	} else {
		uploadedFile.NewFileName = hdr.Filename
		outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName))
	}

	if err != nil {
		return nil, err
	}
	defer outfile.Close()

	fileSize, err := io.Copy(outfile, infile)
	if err != nil {
		return nil, err
	}
	uploadedFile.FileSize = uint64(fileSize)

	return &uploadedFile, nil
}

func (t *Tools) isAllowedType(fileType string) bool {
	for _, allowedType := range t.AllowedTypes {
		if strings.EqualFold(fileType, allowedType) {
			return true
		}
	}
	return false
}

// CreateDirIfNotExist cria um diretório se ele não existir.
func (t *Tools) CreateDirIfNotExist(dir string) error {
	const mode = 0755

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return err
		}
	}

	return nil
}

// Slugify cria um slug seguro para URL a partir de uma string.
func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string not permitted")
	}

	var re = regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(slug) == 0 {
		return "", errors.New("after removing characters, slug is zero length")
	}
	return slug, nil
}

// DownloadStaticFile efetua o download de um arquivo estático, garantindo que o arquivo não seja um diretório
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, path, fileName, displayName string) {
	filePath := filepath.Join(path, fileName)

	// Verifica se o caminho é um diretório. Se for, não serve o arquivo.
	fileInfo, err := os.Stat(filePath)
	if err == nil && fileInfo.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Tenta abrir o arquivo para verificar se há erros de permissão antes de servir.
	// Isso torna o tratamento de erros mais explícito, especialmente para casos de bloqueio de arquivo no Windows.
	f, err := os.Open(filePath)
	if err != nil {
		// Se o erro for de permissão, retorna 403. Para outros erros (como "não encontrado"),
		// deixamos http.ServeFile lidar para garantir o código de status correto (404).
		if os.IsPermission(err) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else {
		// Fecha o arquivo apenas se ele foi aberto com sucesso.
		f.Close()
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(w, r, filePath)
}

type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// WriteJSON efetua a leitura de um JSON, valida se eh um JSON valido,
// comparando com a interface de destino dos dados, e retorna erros detalhados
// em caso de falha na leitura ou validação.
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024 // 1 MB
	if t.MaxJSONSize > 0 {
		maxBytes = t.MaxJSONSize
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}
	err := dec.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
			case errors.As(err, &syntaxError):
				return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
			case errors.Is(err, io.ErrUnexpectedEOF):
				return errors.New("body contains badly-formed JSON")
			case errors.As(err, &unmarshalTypeError):
				if unmarshalTypeError.Field != "" {
					return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
				}
				return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
			case errors.Is(err, io.EOF):
				return errors.New("body must not be empty")
			case strings.HasPrefix(err.Error(), "json: unknown field "):
				fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
				return fmt.Errorf("body contains unknown field %s", fieldName)
			case err.Error() == "http: request body too large":
				return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
			case errors.As(err, &invalidUnmarshalError):
				return fmt.Errorf("internal error: %v", err)
			default:
				return err
		}
	}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

// WriteJSON recebe uma interface, converte para JSON e escreve no response writer.
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil
}