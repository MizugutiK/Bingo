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
var clientsMu sync.Mutex
var clients []*websocket.Conn

// クライアントにメッセージを送信するためのチャネル
var broadcast = make(chan []int)

// ルーム管理のためのインスタンス
var roomManager = NewRoomManager()

// 現在のWebSocket接続を保持する変数
var ws *websocket.Conn

// 数字生成の間隔を保持するグローバル変数
var intervalSeconds int = 5

var numberGenerator = NewNumberGenerator()

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
		http.Error(w, "WebSocket upgrade error", http.StatusInternalServerError)
		return
	}
	defer ws.Close()

	// 接続されたクライアントを追加
	clientsMu.Lock()
	clients = append(clients, ws)
	clientsMu.Unlock()

	// 現在の接続数をログに出力
	log.Printf("新しい WebSocket 接続が確立: %s, 現在の接続数: %d", ws.RemoteAddr(), len(clients))

	// クライアントからのメッセージを読み取る（ここでは使用しない）
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("接続切れた: %v", err)
			// 接続が切れたらクライアントを削除
			clientsMu.Lock()
			for i, client := range clients {
				if client == ws {
					clients = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			clientsMu.Unlock()

			// 現在の接続数をログに出力
			log.Printf("クライアントが切断されました。現在の接続数: %d", len(clients))

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
		numbers := <-broadcast

		for _, client := range clients {
			err := client.WriteJSON(numbers)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				// 接続が切れたらクライアントを削除
				clientsMu.Lock()
				for i, c := range clients {
					if c == client {
						log.Printf("クライアントを削除: %s", client.RemoteAddr())
						clients = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				clientsMu.Unlock()
			}
			// 送信データをログに出力
			log.Printf("Sent data: %v", numbers)
		}
	}
}

// 数字を生成してブロードキャストする関数
func generateNumbers() {
	for {
		// 新しい数字を生成
		newNumber := rand.Intn(75) + 1

		// 生成されたことのない数字を探す
		for numberGenerator.Contains(newNumber) {
			newNumber = rand.Intn(75) + 1
		}

		// 数字を保存
		numberGenerator.AddNumber(newNumber)

		// 生成された数字のリスト全体をクライアントにブロードキャスト
		broadcast <- numberGenerator.GetNumbers()

		// 生成された数字をログに出力
		log.Printf("Sent data: [%d]", newNumber)

		// 数字生成の間隔待ち
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
}

type Room struct {
	ID              string
	Host            string
	Type            string
	Password        string // Added Password field
	Clients         map[*websocket.Conn]bool
	Mutex           sync.Mutex
	BingoGame       *BingoGame
	UserBingoCards  map[*websocket.Conn]BingoCard
	BingoCardMarked map[*websocket.Conn][5][5]bool
	BingoCard       BingoCard
}

// BingoGame 構造体を追加して、ビンゴゲームの状態を管理します
type BingoGame struct {
	GeneratedNumbers []int // 生成された数字のリスト
	IsBingo          bool  // ビンゴが達成されたかどうか
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

// JoinRoomHandler 関数内でビンゴゲームの状態をクライアントに送信
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Join the room
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

	// Add the client to the room
	success := roomManager.JoinRoom(roomID, ws)
	if !success {
		log.Printf("Failed to join room: %s", roomID)
		http.Error(w, "Failed to join room", http.StatusInternalServerError)
		return
	}

	// Broadcast the bingo game state to the client
	broadcastGameState(room)

	// Response
	resp := map[string]string{
		"room_id": roomID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateRoomHandler関数内でルーム作成時に暗証番号を生成
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host     string `json:"host"`
		RoomType string `json:"room_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ルームの作成
	roomID := roomManager.CreateRoom(req.Host, req.RoomType)

	// ルームの初期化
	room := roomManager.Rooms[roomID]
	startNewGame(room) // ビンゴゲームを開始

	// レスポンスを返す
	resp := map[string]string{
		"room_id": roomID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding: %v", err)
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
	numberGenerator.Reset()
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

// generateBingoCard 関数の最後にリターンステートメントを追加する
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

	return card // 追加
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

type NumberGenerator struct {
	numbers []int
	mutex   sync.Mutex
}

// 新しい NumberGenerator を作成する関数
func NewNumberGenerator() *NumberGenerator {
	return &NumberGenerator{
		numbers: make([]int, 0),
	}
}

// 数字を追加するメソッド
func (ng *NumberGenerator) AddNumber(number int) {
	ng.mutex.Lock()
	defer ng.mutex.Unlock()
	ng.numbers = append(ng.numbers, number)
}

// 数字が含まれているかチェックするメソッド
func (ng *NumberGenerator) Contains(number int) bool {
	ng.mutex.Lock()
	defer ng.mutex.Unlock()
	for _, n := range ng.numbers {
		if n == number {
			return true
		}
	}
	return false
}

// 生成された数字のリストを取得するメソッド
func (ng *NumberGenerator) GetNumbers() []int {
	ng.mutex.Lock()
	defer ng.mutex.Unlock()
	return ng.numbers
}

// 数字リストをリセットするメソッド
func (ng *NumberGenerator) Reset() {
	ng.mutex.Lock()
	defer ng.mutex.Unlock()
	ng.numbers = make([]int, 0)
}

// 新しいゲームを開始する関数
func startNewGame(room *Room) {
	// ビンゴカードを生成
	room.BingoCard = generateBingoCard()

	// ビンゴゲームの状態をリセット
	room.BingoGame = &BingoGame{
		GeneratedNumbers: make([]int, 0),
		IsBingo:          false,
	}

	// 新しいゲームの開始をクライアントに通知
	broadcastGameState(room)
}

// 各ルームに参加しているクライアントに、そのルームのビンゴゲームの状態を送信する関数
func broadcastGameState(room *Room) {
	// ビンゴゲームの状態をクライアントに送信
	gameState := map[string]interface{}{
		"bingo_card":        room.BingoCard,
		"bingo_card_marked": room.BingoCardMarked,
		"generated_numbers": room.BingoGame.GeneratedNumbers,
		"is_bingo":          room.BingoGame.IsBingo,
	}

	for client := range room.Clients {
		err := client.WriteJSON(gameState)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			client.Close()
			delete(room.Clients, client)
		}
	}
}

// ユーザーごとのビンゴカードを生成し、部屋に参加している各ユーザーに割り当てます
func generateUserBingoCard(room *Room, ws *websocket.Conn) {
	rand.Seed(time.Now().UnixNano())

	var card BingoCard
	usedNumbers := make(map[int]bool)

	// Generate a 5x5 bingo card
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			num := rand.Intn(75) + 1
			// Avoid duplicate numbers
			for usedNumbers[num] {
				num = rand.Intn(75) + 1
			}
			usedNumbers[num] = true
			card[i][j] = num
		}
	}

	// Middle cell is FREE
	card[2][2] = 0

	// Update user bingo cards for the room
	room.UserBingoCards[ws] = card
	room.BingoCardMarked[ws] = [5][5]bool{} // Initialize marked state as well
}
