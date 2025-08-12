package test

import (
	"bytes"
	"github.com/maximsavonin/Tests/workmate/first/internal"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDownloadAndZip(t *testing.T) {
	limiter := internal.NewRateLimiter(3)
	limiterdownload := internal.NewRateLimiter(3)
	downloadHandler := internal.NewHandler(limiter, limiterdownload)

	ts := httptest.NewServer(http.HandlerFunc(downloadHandler.DownloadAndZip))
	defer ts.Close()

	requestBody := bytes.NewBufferString(`{"urls": ["https://i.pinimg.com/236x/c8/cc/24/c8cc24bba37a25c009647b8875aae0e3.jpg"]}`)

	req, err := http.NewRequest("POST", ts.URL, requestBody)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Logf("Status: %d", resp.StatusCode)

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/zip" {
		t.Fatalf("Content-Type: %s", contentType)
	}

	outputPath := filepath.Join("./files", "downloaded.zip")
	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("zip saved")
}

func TestLimiter(t *testing.T) {
	limiter := internal.NewRateLimiter(3)
	limiterdownload := internal.NewRateLimiter(3)
	downloadHandler := internal.NewHandler(limiter, limiterdownload)

	ts := httptest.NewServer(http.HandlerFunc(downloadHandler.DownloadAndZip))
	defer ts.Close()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			requestBody := bytes.NewBufferString(`{"urls": ["https://i.pinimg.com/236x/c8/cc/24/c8cc24bba37a25c009647b8875aae0e3.jpg"]}`)

			req, err := http.NewRequest("POST", ts.URL, requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			t.Logf("Status: %d", resp.StatusCode)

			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/zip" {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("Content-Type: %s\nResponse: %s", contentType, string(body))
				return
			}

			outputPath := filepath.Join("./files", "downloaded.zip")
			outFile, err := os.Create(outputPath)
			if err != nil {
				t.Fatal(err)
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("zip saved")
		}()
	}

	wg.Wait()
}

func TestCreateZip(t *testing.T) {
	limiter := internal.NewRateLimiter(3)
	limiterdownload := internal.NewRateLimiter(3)
	downloadHandler := internal.NewHandler(limiter, limiterdownload)

	ts := httptest.NewServer(http.HandlerFunc(downloadHandler.CreateZip))
	defer ts.Close()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			requestBody := bytes.NewBufferString(`{"filename": "test"}`)

			req, err := http.NewRequest("POST", ts.URL, requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			t.Logf("Status: %d\nResponse: %s", resp.StatusCode, string(body))
		}()
	}

	wg.Wait()
}

func TestAddToZip(t *testing.T) {
	limiter := internal.NewRateLimiter(3)
	limiterdownload := internal.NewRateLimiter(3)
	downloadHandler := internal.NewHandler(limiter, limiterdownload)

	router := http.NewServeMux()
	router.HandleFunc("/create", downloadHandler.CreateZip)
	router.HandleFunc("/add", downloadHandler.AddToZip)

	// Создаем тестовый сервер с роутером
	ts := httptest.NewServer(router)
	defer ts.Close()

	req, err := http.NewRequest("POST", ts.URL+"/create", bytes.NewBufferString(`{"filename": "test"}`))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			requestBody := bytes.NewBufferString(`{"filename": "test", "urls": ["https://i.pinimg.com/236x/c8/cc/24/c8cc24bba37a25c009647b8875aae0e3.jpg"]}`)

			req, err := http.NewRequest("POST", ts.URL+"/add", requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			t.Logf("Status: %d\nResponse: %s", resp.StatusCode, string(body))
		}()
	}

	wg.Wait()
}
