package internal

// Request структура для входящего JSON
type DownloadRequest struct {
	URLs []string `json:"urls"`
}

// Response структура для ответа об ошибках
type ErrorResponse struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

// DownloadResult содержит результат скачивания файла
type DownloadResult struct {
	URL      string
	Filename string
	Content  []byte
	Error    error
}
