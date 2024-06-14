package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketのアップグレーダー設定
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocket接続しているクライアントを保持するマップ
var clients = make(map[*websocket.Conn]bool)

// 生成されたビンゴの数字を保持するためのリスト
var generatedNumbers = make([]int, 0)

// クライアントにメッセージを送信するためのチャネル
var broadcast = make(chan []int)

// ルーム管理のためのインスタンス
var roomManager = NewRoomManager()

// 現在のWebSocket接続を保持する変数
var ws *websocket.Conn

// 数字生成の間隔を保持するグローバル変数
var intervalSeconds = 5

func main() {
	// 静的ファイルの配信
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	// WebSocketエンドポイント
	http.HandleFunc("/ws", handleConnections)

	// ルーム作成エンドポイント
	http.HandleFunc("/create-room", CreateRoomHandler)

	// ルームに参加するエンドポイント
	http.HandleFunc("/join-room", JoinRoomHandler)

	// 新しいゲームを開始するエンドポイント
	http.HandleFunc("/new-game", NewGameHandler)

	// ビンゴチェックのエンドポイント
	http.HandleFunc("/check-bingo", CheckBingoHandler)

	// 生成された数字のリストをリセットするエンドポイント
	http.HandleFunc("/reset-generated-numbers", ResetGeneratedNumbersHandler)

	// 数字生成間隔を設定するエンドポイント
	http.HandleFunc("/set-interval", SetIntervalHandler)

	// メッセージのハンドリングゴルーチンを開始
	go handleMessages()

	// 数字の生成ゴルーチンを開始
	go generateNumbers()

	// サーバーの起動
	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// WebSocket接続を処理する関数
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// WebSocketのアップグレード
	ws2, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("WebSocket upgrade error: %v", err)
		http.Error(w, "WebSocket upgrade error", http.StatusInternalServerError)
		return
	}
	ws = ws2
	log.Printf("新しい WebSocket 接続が確立: %s", ws.RemoteAddr())

	// 接続されたクライアントを追加
	clients[ws] = true

	// クライアントからのメッセージを待機するループ
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("接続切れた: %v", err)
			delete(clients, ws)
			break
		}
	}
}

// メッセージを処理するゴルーチン
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
			log.Printf("Sent data: %v", number)
		}
	}
}

// 数字を生成してブロードキャストする関数
func generateNumbers() {
	for {
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
		newNumber := generateUniqueNumber()
		generatedNumbers = append(generatedNumbers, newNumber)
		broadcast <- generatedNumbers
	}
}

// 重複しない数字を生成する関数
func generateUniqueNumber() int {
	newNumber := rand.Intn(75) + 1
	for contains(generatedNumbers, newNumber) {
		newNumber = rand.Intn(75) + 1
	}
	return newNumber
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

// ルームに関する定数と構造体
const (
	PublicRoomType  = "public"
	PrivateRoomType = "private"
	PasswordLength  = 6
)

// Room構造体
type Room struct {
	ID       string
	Host     string
	Type     string
	Password string
	Clients  map[*websocket.Conn]bool
	Mutex    sync.Mutex
}

// RoomManager構造体
type RoomManager struct {
	Rooms map[string]*Room
	Mutex sync.Mutex
}

// 新しいRoomManagerインスタンスを作成
func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

// ルームに参加する関数
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

// パスワード生成関数
func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ルーム作成関数
func (rm *RoomManager) CreateRoom(host string, roomType string) string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	password := generatePassword(PasswordLength)
	room := &Room{
		Host:     host,
		Type:     roomType,
		Password: password,
		Clients:  make(map[*websocket.Conn]bool),
	}

	rm.Rooms[password] = room
	log.Printf("新しいルームが作成されました。Host: %s, ルームID: %s, ルームタイプ: %s", host, password, roomType)
	return password
}

// ルームに参加するためのハンドラー関数
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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
		log.Printf("Invalid password: %s", req.Password)
		http.Error(w, "Invalid password", http.StatusForbidden)
		return
	}

	success := roomManager.JoinRoom(roomID, ws)
	if !success {
		log.Printf("Failed to join room: %s", roomID)
		http.Error(w, "Failed to join room", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"room_id": roomID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ルーム作成ハンドラー関数
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host     string `json:"host"`
		RoomType string `json:"room_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	roomID := roomManager.CreateRoom(req.Host, req.RoomType)
	resp := map[string]string{
		"room_id": roomID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 新しいゲームを開始するハンドラー関数
func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	bingoCard := generateBingoCard()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bingoCard)
}

// ビンゴチェックを行うハンドラー関数
func CheckBingoHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Card   BingoCard  `json:"card"`
		Marked [5][5]bool `json:"marked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	isBingo := checkBingo(req.Card, req.Marked)
	resp := map[string]bool{"bingo": isBingo}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 生成された数字のリストをリセットするハンドラー関数
func ResetGeneratedNumbersHandler(w http.ResponseWriter, r *http.Request) {
	generatedNumbers = []int{}
	response := map[string]string{"message": "Generated numbers have been reset"}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to generate JSON response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// 数字生成間隔を設定するハンドラー関数
func SetIntervalHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Interval int `json:"interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	interval, err := strconv.Atoi(strconv.Itoa(req.Interval))
	if err != nil || interval <= 0 {
		log.Printf("Invalid interval value: %v\n", err)
		http.Error(w, "Invalid interval value", http.StatusBadRequest)
		return
	}

	intervalSeconds = interval
	log.Printf("数字生成間隔が設定されました: %d秒", intervalSeconds)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Interval has been set"))
}

// BingoCard型の定義
type BingoCard [5][5]int

// ビンゴカードを生成する関数
func generateBingoCard() BingoCard {
	rand.Seed(time.Now().UnixNano())

	var card BingoCard
	usedNumbers := make(map[int]bool)

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			num := rand.Intn(75) + 1
			for usedNumbers[num] {
				num = rand.Intn(75) + 1
			}
			usedNumbers[num] = true
			card[i][j] = num
		}
	}

	card[2][2] = 0 // FREE space
	return card
}

// ビンゴをチェックする関数
func checkBingo(card BingoCard, marked [5][5]bool) bool {
	// Horizontal check
	for i := 0; i < 5; i++ {
		if marked[i][0] && marked[i][1] && marked[i][2] && marked[i][3] && marked[i][4] {
			return true
		}
	}

	// Vertical check
	for j := 0; j < 5; j++ {
		if marked[0][j] && marked[1][j] && marked[2][j] && marked[3][j] && marked[4][j] {
			return true
		}
	}

	// Diagonal check (left to right)
	diagonal1 := true
	for i := 0; i < 5; i++ {
		if !marked[i][i] {
			diagonal1 = false
			break
		}
	}
	if diagonal1 {
		return true
	}

	// Diagonal check (right to left)
	diagonal2 := true
	for i := 0; i < 5; i++ {
		if !marked[i][4-i] {
			diagonal2 = false
			break
		}
	}
	return diagonal2
}
