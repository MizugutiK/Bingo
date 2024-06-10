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
var intervalSeconds int = 60

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
	ws = ws2

	if err != nil {
		log.Fatalf("WebSocket upgrade error: %v", err)
		http.Error(w, "handleConnections関数エラー", http.StatusInternalServerError)
		return
	}
	log.Printf("新しい WebSocket 接続が確立: %s", ws.RemoteAddr())

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
			// 送信データをログに出力
			log.Printf("Sent data: %v", number)
		}
	}
}

// 数字を生成してブロードキャストする関数
func generateNumbers() {
	for {
		// 設定された秒数ごとに新しい数字を生成
		time.Sleep(time.Duration(intervalSeconds) * time.Second)

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
	Type     string // ルームのタイプ (public or private)
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
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		log.Printf("Error デコーディング: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ルームの作成
	roomID := roomManager.CreateRoom(req.Host, req.RoomType)

	resp := map[string]string{
		"room_id": roomID,
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

// RoomManager構造体にルームを作成するための関数を追加
func (rm *RoomManager) CreateRoom(host string, roomType string) string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	password := generatePassword(PasswordLength) // パスワードを生成

	room := &Room{
		Host:     host,
		Type:     roomType, // ルームタイプを設定
		Password: password,
		Clients:  make(map[*websocket.Conn]bool),
	}

	// パスワードをキーとしてRoomを登録
	rm.Rooms[password] = room
	// ルームのタイプをログに出力
	log.Printf("新しいルームが作成されました。Host: %s, ルームID: %s, ルームタイプ: %s", host, password, roomType)
	return password
}

// NewGameHandlerは新しいビンゴゲームを開始し、ビンゴカードを生成してクライアントに返します
func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	// ビンゴカードを生成
	bingoCard := generateBingoCard()

	// JSON形式でレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bingoCard)
}

// CheckBingoHandlerはビンゴが達成されたかどうかをチェックします
func CheckBingoHandler(w http.ResponseWriter, r *http.Request) {
	// リクエストボディをデコード
	var req struct {
		Card   BingoCard  `json:"card"`
		Marked [5][5]bool `json:"marked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// ビンゴが達成されたかをチェック
	isBingo := checkBingo(req.Card, req.Marked)

	// レスポンスをJSON形式で返す
	json.NewEncoder(w).Encode(map[string]bool{"bingo": isBingo})
}

// ResetGeneratedNumbersHandlerは生成された数字のリストをリセットします
func ResetGeneratedNumbersHandler(w http.ResponseWriter, r *http.Request) {
	generatedNumbers = []int{}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Generated numbers have been reset"))
}

// SetIntervalHandlerは数字生成間隔を設定するハンドラーです
func SetIntervalHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Interval int `json:"interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 文字列から整数に変換
	interval, err := strconv.Atoi(strconv.Itoa(req.Interval))
	if err != nil {
		log.Printf("Error converting interval to integer: %v\n", err)
		http.Error(w, "Invalid interval value", http.StatusBadRequest)
		return
	}

	if interval <= 0 {
		http.Error(w, "Invalid interval value", http.StatusBadRequest)
		return
	}
	intervalSeconds = interval
	log.Printf("数字生成間隔が設定されました: %d秒", intervalSeconds)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Interval has been set"))
}

// BingoCardはビンゴカードを表す型です
type BingoCard [5][5]int

// generateBingoCardは新しいビンゴカードを生成します
func generateBingoCard() BingoCard {
	rand.Seed(time.Now().UnixNano())

	var card BingoCard
	usedNumbers := make(map[int]bool)

	// 5x5のビンゴカードを生成
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			num := rand.Intn(75) + 1
			// 重複する数字を避ける
			for usedNumbers[num] {
				num = rand.Intn(75) + 1
			}
			usedNumbers[num] = true
			card[i][j] = num
		}
	}

	// 中央のセルはFREEとする
	card[2][2] = 0
	return card
}

// checkBingoはビンゴが達成されたかどうかをチェックします
func checkBingo(card BingoCard, marked [5][5]bool) bool {
	// 横方向のチェック
	for i := 0; i < 5; i++ {
		if marked[i][0] && marked[i][1] && marked[i][2] && marked[i][3] && marked[i][4] {
			return true
		}
	}

	// 縦方向のチェック
	for j := 0; j < 5; j++ {
		if marked[0][j] && marked[1][j] && marked[2][j] && marked[3][j] && marked[4][j] {
			return true
		}
	}

	// 斜め方向のチェック（左上から右下）
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

	// 斜め方向のチェック（右上から左下）
	diagonal2 := true
	for i := 0; i < 5; i++ {
		if !marked[i][4-i] {
			diagonal2 = false
			break
		}
	}
	return diagonal2
}
