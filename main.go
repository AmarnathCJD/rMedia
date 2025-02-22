package main

import (
	"fmt"
	"log"
	"main/utils"
	"net/http"
	"os"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/joho/godotenv"
)

var client *tg.Client

func main() {
	godotenv.Load()
	client, _ = tg.NewClient(tg.ClientConfig{
		AppID:    2040,
		AppHash:  os.Getenv("APP_HASH"),
		LogLevel: tg.LogInfo,
	})

	client.Conn()
	client.LoginBot(os.Getenv("BOT_TOKEN"))

	client.AddMessageHandler("/fid", utils.GetBotFileID)

	http.HandleFunc("/stream/", streamHandler)
	log.Println("Server running on :80")
	log.Fatal(http.ListenAndServe("0.0.0.0:80", nil))
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	//log.Println("stream-request:", r.URL.Path, r.Header.Get("Range"))
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusTeapot)
		return
	}
	fileID := parts[2]

	fi, err := utils.ResolveBotFileID(fileID)
	if err != nil {
		http.Error(w, `{"error": "Invalid file ID"}`, http.StatusBadRequest)
		return
	}

	fileSize := int(fi.(*tg.MessageMediaDocument).Document.(*tg.DocumentObj).Size)
	const maxChunkSize = 1024 * 1024

	var start, end int

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
		rangeVal := strings.TrimPrefix(rangeHeader, "bytes=")
		ranges := strings.Split(rangeVal, "-")

		if len(ranges) > 0 && ranges[0] != "" {
			if s, e := strconv.Atoi(ranges[0]); e == nil {
				start = s
			}
		}

		if len(ranges) > 1 && ranges[1] != "" {
			if e, e2 := strconv.Atoi(ranges[1]); e2 == nil {
				end = e
			}
		} else {
			end = start + maxChunkSize - 1
		}
	} else {
		start = 0
		end = fileSize - 1
	}

	if end >= fileSize {
		end = fileSize - 1
	}

	if start > end {
		http.Error(w, `{"error": "Invalid range"}`, http.StatusRequestedRangeNotSatisfiable)
		return
	}

	data, realStart, realEnd, err := utils.DlChunk(client, fi, start, end)
	if err != nil {
		http.Error(w, `{"error": "Error fetching file chunks"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "video/x-matroska")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", realStart, realEnd, fileSize))
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusPartialContent)
	w.Write(data)
}
