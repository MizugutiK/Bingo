package main

import (
	"encoding/json"
	"io/ioutil"
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

var ws *websocket.Conn

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
	ws2, err := upgrader.Upgrade(w, r, nil)
	ws = ws2

	if err != nil {
		log.Fatalf("WebSocket upgrade error: %v", err)
		http.Error(w, "handleConnections関数エラー", http.StatusInternalServerError)
	}
	log.Printf("新しい WebSocket 接続が確立: %s", ws.RemoteAddr())

	// defer ws.Close()

	// 接続されたクライアントを追加
	clients[ws] = true

	for {

		// クライアントからのメッセージを読み取る（ここでは使用しない）
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("接続切れた: %v", err)
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
			// 送信データをログに出力
			log.Printf("Sent data: %v", number)
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
	Password string // パスワード
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

// RoomManager構造体にルームを作成するための関数を追加
func (rm *RoomManager) CreateRoom(host string, roomType string) string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	password := generatePassword(PasswordLength) // パスワードを生成

	room := &Room{
		Host:     host,
		IsPublic: roomType == PublicRoomType,
		Password: password, // パブリックまたはプライベートを指定
		Clients:  make(map[*websocket.Conn]bool),
	}
	// パスワードをキーとしてRoomを登録
	rm.Rooms[password] = room
	return password
}

// JoinRoom関数内でルームに参加する際にパスワードを検証する
func (rm *RoomManager) JoinRoom(password string, ws *websocket.Conn) bool {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	room, exists := rm.Rooms[password]
	if !exists {
		return false
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	room.Clients[ws] = true
	return true
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

// JoinRoomHandler 関数内でルームに参加する際にパスワードを検証する
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("JoinRoomHandler 関数通ってる")

	// リクエストボディを読み取る前にログ出力
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error リクエストボディ: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	log.Printf("Request body: %s", string(requestBody))

	var req struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		log.Printf("Error デコーディング: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// パスワードを使用してルームを特定
	var room *Room
	var roomID string
	for id, r := range roomManager.Rooms {
		if r.Password == req.Password {
			room = r
			roomID = id
			break
		}
	}
	if room == nil {
		log.Printf("パスワード: %s", req.Password)
		http.Error(w, "Invalid password", http.StatusForbidden)
		return
	}

	// ルームにクライアントを参加させる
	success := roomManager.JoinRoom(roomID, ws)
	if !success {
		log.Printf("ルーム参加できてない: %s", roomID)
		http.Error(w, "Failed to join room", http.StatusInternalServerError)
		return
	}

	// 参加が成功した場合は、クライアントにルームIDを返す
	resp := map[string]string{
		"room_id": roomID,
	}
	// 正常なレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("JoinRoomHandler関数エラー: JSON encode error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateRoomHandler関数内でルーム作成時に暗証番号を生成
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Host     string `json:"host"`
		RoomType string `json:"room_type"` // ルームのタイプをリクエストボディから取得

	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("CreateRoomHandler関数エラー: JSON decode error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	password := roomManager.CreateRoom(req.Host, req.RoomType)
	log.Printf("CreateRoomHandler関数問題なし. Host: %s, Room ID: %s", req.Host, password)

	// レスポンスにパスワードを含める
	resp := map[string]string{
		"password": password,
	}

	// 正常なレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("CreateRoomHandler関数エラー: JSON encode error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
