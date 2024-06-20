package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
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

// RoomManager構造体
type RoomManager struct {
	Rooms map[string]*Room
	Mutex sync.Mutex
}

// Room構造体
type Room struct {
	Password string
	Clients  map[*websocket.Conn]bool
	Mutex    sync.Mutex
}

// レスポンス用の構造体
type ResponseData struct {
	Numbers []int `json:"numbers"`
}

// 新しいRoomManagerインスタンスを作成
func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

// WebSocket接続を処理する関数
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// WebSocketのアップグレード
	ws2, err := upgrader.Upgrade(w, r, nil)
	ws = ws2
	if err != nil {
		log.Fatalf("handleConnections関数WebSocket アップグレード エラー: %v", err)
		http.Error(w, "WebSocket アップグレード エラー", http.StatusInternalServerError)
		return
	}
	defer ws.Close()
	// 初回メッセージでルーム名とパスワードを受け取る
	var req struct {
		Password string `json:"password"`
	}
	if err := ws.ReadJSON(&req); err != nil {
		log.Printf("handleConnections関数初回メッセージの読み取りエラー: %v", err)

		return
	}

	// ルームに参加
	success := roomManager.JoinRoom(req.Password, ws)
	if !success {
		log.Printf("handleConnections関数部屋に参加できませんでした: %s", req.Password)

		return
	}

	log.Printf("新しい WebSocket 接続が確立: %s", ws.RemoteAddr())

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

// JoinRoom ルームに参加する関数
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

// CreateRoomHandler 部屋を作成するハンドラー関数
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	password := roomManager.CreateRoom()
	if password == "" {
		log.Println("部屋の作成に失敗しました")
		http.Error(w, "部屋の作成に失敗しました", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"password": password,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ルームごとの数字を取得するハンドラー関数
func GetRoomNumbersHandler(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get("password")
	numbers, err := roomManager.GetNumbersForRoom(password)
	if err != nil {
		log.Printf("数字の取得に失敗しました: %v", err)
		http.Error(w, "数字の取得に失敗しました", http.StatusInternalServerError)
		return
	}

	resp := ResponseData{
		Numbers: numbers,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ルームの情報からファイル名を生成する関数
func getFileName(room *Room) string {
	return fmt.Sprintf("%s.txt", room.Password)
}

// テキストファイルから数字を読み取る関数
func readNumbersFromFile(fileName string) ([]int, error) {
	var numbers []int

	file, err := os.Open(fileName)
	if err != nil {
		return numbers, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		num, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return numbers, err
		}
		numbers = append(numbers, num)
	}

	return numbers, nil
}

// ルーム作成関数
func (rm *RoomManager) CreateRoom() string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	password := generatePassword(PasswordLength)
	room := &Room{
		Password: password,
		Clients:  make(map[*websocket.Conn]bool),
	}

	rm.Rooms[password] = room

	log.Printf("新しいルームが作成されました. Password: %s", password)

	return password
}

func generateAndWriteNumbersToFiles() {
	for {
		// ルームが存在しない場合は待機する
		if len(roomManager.Rooms) == 0 {
			log.Println("ルームが存在しないため、数字の生成を待機しています...")
			time.Sleep(time.Second * 10) // 10秒待機して再試行する
			continue
		}

		// すべてのルームのファイルに数字を書き込む
		for _, room := range roomManager.Rooms {
			newNumber := generateUniqueNumber() // 重複しない数字を生成する
			fileName := getFileName(room)       // ルームごとのファイル名を取得する

			// ファイルをオープン（追記モードで、存在しない場合は作成）
			file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				log.Printf("ファイル %s のオープンに失敗しました: %v", fileName, err)
				continue
			}

			// ファイルに新しい数字を書き込む
			if _, err := file.WriteString(fmt.Sprintf("%d\n", newNumber)); err != nil {
				log.Printf("ファイル %s への書き込みに失敗しました: %v", fileName, err)
			}

			// ファイルをクローズする
			if err := file.Close(); err != nil {
				log.Printf("ファイル %s のクローズに失敗しました: %v", fileName, err)
			}
		}

		// 一定時間待機する（例として1秒）
		time.Sleep(1000)
	}
}

// GetNumbersForRoomメソッドを定義
func (rm *RoomManager) GetNumbersForRoom(password string) ([]int, error) {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	room := rm.Rooms[password]
	if room == nil {
		return nil, fmt.Errorf("ルームが見つかりません: %s", password)
	}

	// ファイル名を取得
	fileName := getFileName(room)

	// テキストファイルから数字を読み取る
	numbers, err := readNumbersFromFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("数字の読み取りに失敗しました: %v", err)
	}

	return numbers, nil
}

// 重複しない数字を生成する関数
func generateUniqueNumber() int {
	for {
		newNumber := rand.Intn(75) + 1
		if !contains(generatedNumbers, newNumber) {
			generatedNumbers = append(generatedNumbers, newNumber)
			// log.Printf("生成された数字: %d", newNumber) // ログ出力
			return newNumber
		}
	}
}

// スライスに指定された値が含まれているかを確認する関数
func contains(slice []int, item int) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
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

// WebSocket接続しているクライアントを保持するマップ
var clients = make(map[*websocket.Conn]bool)

// ルーム管理のためのインスタンス
var roomManager = NewRoomManager()

// 現在のWebSocket接続を保持する変数
var ws *websocket.Conn

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
	// ルームごとの数字取得エンドポイント
	http.HandleFunc("/get-room-numbers", GetRoomNumbersHandler)

	// サーバーの起動
	log.Println("Listening on :8080...")
	go generateAndWriteNumbersToFiles() // 数字生成のゴルーチンを起動
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// HTTPハンドラー
func handleNumbersRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ファイル名を構成
	fileName := fmt.Sprintf("%s.txt", req.Password)

	// テキストファイルから数字を読み取る
	numbers, err := readNumbersFromFile(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// レスポンスデータを構築
	responseData := ResponseData{
		Numbers: numbers,
	}

	// JSON形式でレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

var generatedNumbers []int // 重複をチェックするためのスライス

// ルームに関する定数と構造体
const (
	PasswordLength = 6
)

// パスワードに基づいてルームを取得する関数
func (rm *RoomManager) GetRoomByPassword(password string) *Room {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	return rm.Rooms[password]
}

func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	// ルームを作成する
	password := roomManager.CreateRoom()
	if password == "" {
		log.Printf("JoinRoomHandler関数部屋の作成に失敗しました")
		http.Error(w, "部屋の作成に失敗しました", http.StatusInternalServerError)
		return
	}

	// レスポンスデータを構造体に格納
	response := struct {
		Card     [][]interface{} `json:"card"`
		Interval int             `json:"interval"`
	}{

		Interval: 10, // 例として10秒のインターバルを設定
	}

	// レスポンスをJSON形式で送信
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JoinRoomHandler関数レスポンスのエンコードエラー: %v", err)
		http.Error(w, "レスポンスのエンコードに失敗しました", http.StatusInternalServerError)
		return
	}
}

// ビンゴカードを生成するハンドラー関数
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
		log.Printf("リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエスト本文が無効です", http.StatusBadRequest)
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
	response := map[string]string{"message": "生成された番号はリセットされました"}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "JSON 応答の生成に失敗しました", http.StatusInternalServerError)
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
		log.Printf("SetIntervalHandler関数リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエスト本文が無効です", http.StatusBadRequest)
		return
	}

	interval, err := strconv.Atoi(strconv.Itoa(req.Interval))
	if err != nil || interval <= 0 {
		log.Printf("SetIntervalHandler関数 無効な間隔値: %v\n", err)
		http.Error(w, "無効な間隔値", http.StatusBadRequest)
		return
	}

	// intervalSeconds = interval
	// log.Printf("数字生成間隔が設定されました: %d秒", intervalSeconds)
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
