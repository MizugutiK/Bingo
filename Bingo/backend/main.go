package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
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
var generatedNumbers = make([]int, 0)

// クライアントにメッセージを送信するためのチャネル
var broadcast = make(chan []int)

// ルーム管理
var roomManager = NewRoomManager()

func main() {
	// 静的ファイルの配信
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// WebSocketエンドポイント
	http.HandleFunc("/ws", handleConnections)

	// ルーム作成エンドポイント
	http.HandleFunc("/create-room", CreateRoomHandler)

	// ルームに参加するエンドポイント
	http.HandleFunc("/join-room", JoinRoomHandler)

	// 新しいゲームを開始するエンドポイント
	http.HandleFunc("/new-game", NewGameHandler)

	// ハンドラーを登録
	http.HandleFunc("/check-bingo", CheckBingoHandler)

	// 生成された数字のリストをリセットするエンドポイント
	http.HandleFunc("/reset-generated-numbers", ResetGeneratedNumbersHandler)

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

// ルームに関する定数
const (
	PublicRoomType  = "public"
	PrivateRoomType = "private"
	PasswordLength  = 6 // 暗証番号の文字数
)

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
		// time.Sleep(10 * time.Second)
		time.Sleep(1 * time.Minute)

		// 新しい数字を生成
		newNumber := rand.Intn(75) + 1

		// 生成されたことのない数字を探す
		for contains(generatedNumbers, newNumber) {
			newNumber = rand.Intn(75) + 1
		}

		// 数字を保存
		generatedNumbers = append(generatedNumbers, newNumber)

		// 生成された数字のリスト全体をクライアントにブロードキャスト
		broadcast <- generatedNumbers
	}
}

// 指定された数字がリストに含まれているかどうかを確認する関数
func contains(numbers []int, number int) bool {
	for _, n := range numbers {
		if n == number {
			return true
		}
	}
	return false
}

// Room構造体に暗証番号を追加
type Room struct {
	ID       string
	Host     string
	IsPublic bool
	Password string // 暗証番号
	Clients  map[*websocket.Conn]bool
	Mutex    sync.Mutex
}
type RoomManager struct {
	Rooms map[string]*Room
	Mutex sync.Mutex
}

// NewRoomManager関数内でRoom構造体にPasswordを追加
func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

// CreateRoom関数内で暗証番号を生成してRoom構造体に追加
func (rm *RoomManager) CreateRoom(host string, roomType string) string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	roomID := generateRoomID()
	password := generatePassword(PasswordLength) // 暗証番号を生成

	room := &Room{
		ID:       roomID,
		Host:     host,
		IsPublic: roomType == PublicRoomType,
		Password: password, // パブリックまたはプライベートを指定
		Clients:  make(map[*websocket.Conn]bool),
	}
	rm.Rooms[roomID] = room
	return roomID
}

func (rm *RoomManager) JoinRoom(roomID string, ws *websocket.Conn) bool {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	room, exists := rm.Rooms[roomID]
	if !exists {
		return false
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	room.Clients[ws] = true
	return true
}

// CreateRoomHandler関数内でルーム作成時に暗証番号を生成
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host     string `json:"host"`
		RoomType string `json:"room_type"` // ルームのタイプをリクエストボディから取得
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roomID := roomManager.CreateRoom(req.Host, req.RoomType)
	room := roomManager.Rooms[roomID] // ルームを取得

	// 暗証番号を生成
	password := generatePassword(PasswordLength)

	// ルームに暗証番号を設定
	room.Password = password

	// ログを追加して暗証番号が正しく設定されたか確認
	log.Printf("Room created: ID=%s, Password=%s", roomID, password)

	// レスポンスにルームIDと暗証番号を含める
	resp := map[string]string{
		"room_id":  roomID,
		"password": password, // ここで生成した暗証番号を使用する
	}
	json.NewEncoder(w).Encode(resp)

	// 追加：クライアントに返すデータの形式を確認するためのログ
	log.Printf("Response sent to client: %v", resp)
}

// generatePassword関数を追加して暗証番号を生成する
func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// JoinRoomHandlerはルームに参加するハンドラです
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomID string `json:"room_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// WebSocketのアップグレード
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	success := roomManager.JoinRoom(req.RoomID, ws)
	if !success {
		http.Error(w, "Failed to join room", http.StatusNotFound)
		ws.Close()
		return
	}

	// ルームに参加する際に暗証番号もクライアントに送信する
	room := roomManager.Rooms[req.RoomID]
	resp := map[string]string{
		"room_id":  req.RoomID,
		"password": room.Password,
	}
	// WebSocketでクライアントにメッセージを送信する
	err = ws.WriteJSON(resp)
	if err != nil {
		log.Printf("WebSocket write error: %v", err)
		ws.Close()
		return
	}

	// 処理したWebSocket接続をログに記録
	log.Printf("Client joined room: %s", req.RoomID)
}

// generateRoomIDはランダムな文字列を生成して返す関数です
func generateRoomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
