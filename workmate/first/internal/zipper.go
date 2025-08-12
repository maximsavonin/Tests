package internal

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Создаём обработчик
func NewHandler(limiter *RateLimiter, limiterdownload *RateLimiter) *Handler {
	return &Handler{limiter: limiter, limiterdownload: limiterdownload}
}

// Скачиваем архивируем и сразу возвращаем zip
func (h *Handler) DownloadAndZip(w http.ResponseWriter, r *http.Request) {
	err := h.limiter.TryAcquire()
	if err != nil {
		http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		return
	}
	defer h.limiter.Release()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим входящий JSON
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	filename := time.Now().Format("20060102_1504") + ".zip"
	if req.FileName != "" {
		filename = req.FileName
		if !strings.HasSuffix(filename, ".zip") {
			filename += ".zip"
		}
	}

	// Скачиваем файлы параллельно
	results := h.downloadFiles(req.URLs)

	// Создаем ZIP архив в памяти
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	var errors []ErrorResponse
	var hasSuccess bool

	// Добавляем файлы в архив
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, ErrorResponse{
				URL:   result.URL,
				Error: result.Error.Error(),
			})
			continue
		}

		// Создаем файл в архиве
		writer, err := zipWriter.Create(result.Filename)
		if err != nil {
			errors = append(errors, ErrorResponse{
				URL:   result.URL,
				Error: fmt.Sprintf("failed to add to zip: %v", err),
			})
			continue
		}

		// Копируем содержимое
		if _, err := io.Copy(writer, bytes.NewReader(result.Content)); err != nil {
			errors = append(errors, ErrorResponse{
				URL:   result.URL,
				Error: fmt.Sprintf("failed to write to zip: %v", err),
			})
			continue
		}

		hasSuccess = true
	}

	// Закрываем архив
	if err := zipWriter.Close(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create zip: %v", err), http.StatusInternalServerError)
		return
	}

	// Если ни один файл не скачался
	if !hasSuccess {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPartialContent)
		json.NewEncoder(w).Encode(errors)
		return
	}

	// Отправляем архив или ошибки
	if len(errors) > 0 {
		w.Header().Set("X-Errors", "true")
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	io.Copy(w, zipBuffer)
}

// Создаём архив
func (h *Handler) CreateZip(w http.ResponseWriter, r *http.Request) {
	err := h.limiter.TryAcquire()
	if err != nil {
		http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		return
	}
	defer h.limiter.Release()

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		http.Error(w, "Error file name", http.StatusBadRequest)
		return
	}

	filename := req.FileName
	if !strings.HasSuffix(filename, ".zip") {
		filename += ".zip"
	}

	if _, err = os.Open(filename); err == nil {
		http.Error(w, "Error file name", http.StatusBadRequest)
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Error create zip", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"filename": filename,
	})
}

// Добавляем файлы в архив
func (h *Handler) AddToZip(w http.ResponseWriter, r *http.Request) {
	err := h.limiter.TryAcquire()
	if err != nil {
		http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		return
	}
	defer h.limiter.Release()

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		http.Error(w, "Error file name", http.StatusBadRequest)
		return
	}

	filename := req.FileName
	if !strings.HasSuffix(filename, ".zip") {
		filename += ".zip"
	}

	// открываем архив
	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, "Error not such file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Парсим входящий JSON
	if len(req.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	// Скачиваем файлы параллельно
	results := h.downloadFiles(req.URLs)

	var errors []ErrorResponse
	var hasSuccess bool

	// Добавляем файлы в архив
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, ErrorResponse{
				URL:   result.URL,
				Error: result.Error.Error(),
			})
			continue
		}

		// Создаем файл в архиве
		writer, err := zipWriter.Create(result.Filename)
		if err != nil {
			errors = append(errors, ErrorResponse{
				URL:   result.URL,
				Error: fmt.Sprintf("failed to add to zip: %v", err),
			})
			continue
		}

		// Копируем содержимое
		if _, err := io.Copy(writer, bytes.NewReader(result.Content)); err != nil {
			errors = append(errors, ErrorResponse{
				URL:   result.URL,
				Error: fmt.Sprintf("failed to write to zip: %v", err),
			})
			continue
		}

		hasSuccess = true
	}

	// Если ни один файл не скачался
	if !hasSuccess {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPartialContent)
		json.NewEncoder(w).Encode(errors)
		return
	}

	if len(errors) > 0 {
		w.Header().Set("X-Errors", "true")
	}
}

// Отправляем архив
func (h *Handler) DownloadZip(w http.ResponseWriter, r *http.Request) {
	err := h.limiter.TryAcquire()
	if err != nil {
		http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		return
	}
	defer h.limiter.Release()

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		http.Error(w, "Error file name", http.StatusBadRequest)
		return
	}

	filename := req.FileName
	if !strings.HasSuffix(filename, ".zip") {
		filename += ".zip"
	}

	// открываем архив
	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, "Error not such file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Получаем информацию о файле
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "File error", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=archive.zip")
	w.Header().Set("Content-Length", fmt.Sprint(fileInfo.Size()))

	// Потоковая отправка (экономит память)
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("Download interrupted: %v", err)
		return
	}
}

// Отправляем и удаляем архив
func (h *Handler) DownloadZipAndDelete(w http.ResponseWriter, r *http.Request) {
	err := h.limiter.TryAcquire()
	if err != nil {
		http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		return
	}
	defer h.limiter.Release()

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		http.Error(w, "Error file name", http.StatusBadRequest)
		return
	}

	filename := req.FileName
	if !strings.HasSuffix(filename, ".zip") {
		filename += ".zip"
	}

	// открываем архив
	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, "Error not such file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Получаем информацию о файле
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "File error", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=archive.zip")
	w.Header().Set("Content-Length", fmt.Sprint(fileInfo.Size()))

	// Потоковая отправка (экономит память)
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("Download interrupted: %v", err)
		return
	}

	// Удаление файла ПОСЛЕ успешной отправки
	err = os.Remove(filename)
	if err != nil {
		log.Printf("Failed to delete file: %v", err)
	}
}

// Скачиваем все файлы по ссылкам
func (h *Handler) downloadFiles(urls []string) []DownloadResult {
	var wg sync.WaitGroup
	results := make([]DownloadResult, len(urls))
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: nil, // Отключаем прокси
		},
	}

	for i, urln := range urls {
		wg.Add(1)
		go func(i int, urln string) {
			h.limiterdownload.Acquire()
			defer h.limiterdownload.Release()

			defer wg.Done()
			result := DownloadResult{URL: urln}

			// Валидация URL
			if _, err := url.ParseRequestURI(urln); err != nil {
				result.Error = fmt.Errorf("invalid URL")
				results[i] = result
				return
			}

			// Скачивание файла
			resp, err := client.Get(urln)
			if err != nil {
				result.Error = fmt.Errorf("download failed: %v", err)
				results[i] = result
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				result.Error = fmt.Errorf("server returned: %s", resp.Status)
				results[i] = result
				return
			}

			// Чтение содержимого
			content, err := io.ReadAll(resp.Body)
			if err != nil {
				result.Error = fmt.Errorf("failed to read content: %v", err)
				results[i] = result
				return
			}

			// Определение имени файла
			filename := filepath.Base(urln)
			if filename == "." || filename == "/" {
				filename = fmt.Sprintf("file_%d", i)
				if contentType := resp.Header.Get("Content-Type"); contentType != "" {
					ext := getContentType(contentType)
					if ext != "" {
						filename += ext
					}
				}
			}

			result.Filename = handleFilename(filename)
			result.Content = content
			results[i] = result
		}(i, urln)
	}

	wg.Wait()
	return results
}

// Удаляем небезопасные символы
func handleFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	return filename
}

// получаем тип файла
func getContentType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// удаление файлов которые не изменялись более 2 часов
func FileDeleter() {
	for true {
		files, err := os.ReadDir("./")
		if err != nil {
			fmt.Printf("Error read dir: %s", err)
		}

		for _, file := range files {
			if file.IsDir() || !strings.EqualFold(filepath.Ext(file.Name()), ".zip") {
				continue
			}

			filePath := filepath.Join("./", file.Name())
			fileInfo, err := file.Info()
			if err != nil {
				fmt.Printf("Error read Info: %s", err)
				continue
			}

			if time.Since(fileInfo.ModTime()) > 2*time.Hour {
				fmt.Printf("File deletet: %s (no modify %v)\n",
					file.Name(), time.Since(fileInfo.ModTime()))
				os.Remove(filePath)
			}
		}
		time.Sleep(time.Minute)
	}
}
