package internal

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DownloadHandler struct {
	limiter *RateLimiter
}

func NewDownloadHandler(limiter *RateLimiter) *DownloadHandler {
	return &DownloadHandler{limiter: limiter}
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.limiter.Acquire()
	defer h.limiter.Release()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим входящий JSON
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	// Скачиваем файлы параллельно
	results := downloadFiles(req.URLs)

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
	w.Header().Set("Content-Disposition", "attachment; filename=download.zip")
	io.Copy(w, zipBuffer)
}

func downloadFiles(urls []string) []DownloadResult {
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
					ext := getExtensionFromMIME(contentType)
					if ext != "" {
						filename += ext
					}
				}
			}

			result.Filename = sanitizeFilename(filename)
			result.Content = content
			results[i] = result
		}(i, urln)
	}

	wg.Wait()
	return results
}

func sanitizeFilename(filename string) string {
	// Удаляем небезопасные символы
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	return filename
}

func getExtensionFromMIME(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
}
