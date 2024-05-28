package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketのアップグレーダー
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocket接続しているクライアント
var clients = make(map[*websocket.Conn]bool)

// 生成されたビンゴの数字を保持するためのリスト
var generatedNumbers = make(map[int]bool)

// クライアントにメッセージを送信するためのチャネル
var broadcast = make(chan int)

func main() {
	// 静的ファイルの配信
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// WebSocketエンドポイント
	http.HandleFunc("/ws", handleConnections)

	// 新しいゲームを開始するエンドポイント
	http.HandleFunc("/new-game", NewGameHandler)

	// メッセージのハンドリング
	go handleMessages()

	// 数字の生成
	go generateNumbers()

	// ログ出力
	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// WebSocket接続を処理する
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// WebSocketのアップグレード
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("WebSocket upgrade error: %v", err)
	}
	defer ws.Close()

	// 接続されたクライアントを追加
	clients[ws] = true

	for {
		// クライアントからのメッセージを読み取る（ここでは使用しない）
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			delete(clients, ws)
			break
		}
	}
}

// メッセージを処理する
func handleMessages() {
	for {
		// メッセージを受信し、クライアントにブロードキャストする
		number := <-broadcast
		for client := range clients {
			err := client.WriteJSON(number)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// 数字を生成してブロードキャストする関数
func generateNumbers() {
	for {
		time.Sleep(1 * time.Minute)

		// 新しい数字を生成
		newNumber := rand.Intn(75) + 1

		// 生成されたことのない数字を探す
		for generatedNumbers[newNumber] {
			newNumber = rand.Intn(75) + 1
		}

		// 数字を保存
		generatedNumbers[newNumber] = true

		// クライアントにブロードキャスト
		for client := range clients {
			err := client.WriteJSON(newNumber)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
