package main

import (
	"first/internal"
	"net/http"
)

func main() {
	// Создаем лимитер на 3 одновременных запросов
	limiter := internal.NewRateLimiter(3)

	// Инициализируем обработчик с лимитером
	downloadHandler := internal.NewDownloadHandler(limiter)

	// Настраиваем маршруты
	http.Handle("/download", downloadHandler)

	// Запускаем сервер
	http.ListenAndServe(":8080", nil)
}
