package toolkit

import (
	"crypto/rand"
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
	MaxFileSize int
	AllowedTypes []string
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