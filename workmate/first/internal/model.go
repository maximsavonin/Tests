package internal

// структура для входящего JSON
type Request struct {
	FileName string   `json:"filename"`
	URLs     []string `json:"urls"`
}

// структура для ответа об ошибках
type ErrorResponse struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

// результат скачивания файла
type DownloadResult struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	Error    error  `json:"error"`
}

// структура для удобного хранения лимитов (типа ООП) для нашего обработчика запросов
type Handler struct {
	limiter         *RateLimiter
	limiterdownload *RateLimiter
}
