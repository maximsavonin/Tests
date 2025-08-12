package main

import (
	"fmt"
	"github.com/maximsavonin/Tests/workmate/first/internal"
	"net/http"
)

func main() {
	// Создаем лимиты на 3 слота
	limiter := internal.NewRateLimiter(3)
	limiterdownload := internal.NewRateLimiter(3)

	// Настраиваем маршруты
	downloadHandler := internal.NewHandler(limiter, limiterdownload)

	http.HandleFunc("/downloadandzip", downloadHandler.DownloadAndZip)
	http.HandleFunc("/createzip", downloadHandler.CreateZip)
	http.HandleFunc("/addtozip", downloadHandler.AddToZip)
	http.HandleFunc("/downloadzip", downloadHandler.DownloadZip)
	http.HandleFunc("/downloadzipanddelete", downloadHandler.DownloadZipAndDelete)

	// Запускаем сервер
	fmt.Println("Server Started")
	http.ListenAndServe(":8080", nil)
}
