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

// WebSocket接続しているクライアントを保持するマップ
var clients = make(map[*websocket.Conn]bool)
var generatedNumbers = make([]int, 0)

// ルーム管理のためのインスタンス
var roomManager = NewRoomManager()

// 現在のWebSocket接続を保持する変数
var ws *websocket.Conn

// // 数字生成の間隔を保持するグローバル変数
// var intervalSeconds = 1

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

	// 初回メッセージでルーム名とパスワードを受け取る
	var req struct {
		RoomName string `json:"room_name"`
		Password string `json:"password"`
	}
	if err := ws.ReadJSON(&req); err != nil {
		log.Printf("handleConnections関数初回メッセージの読み取りエラー: %v", err)
		ws.Close()
		return
	}

	// ルームに参加
	success := roomManager.JoinRoom(req.Password, ws)
	if !success {
		log.Printf("handleConnections関数部屋に参加できませんでした: %s", req.Password)
		ws.Close()
		return
	}

	// 接続されたクライアントを追加
	clients[ws] = true

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

// ルーム作成関数
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomName string `json:"room_name"`
		RoomType string `json:"room_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("CreateRoomHandler関数リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエスト本文が無効です", http.StatusBadRequest)
		return
	}

	// roomManager.CreateRoom はパスワードのみを返す
	password := roomManager.CreateRoom(req.RoomName, req.RoomType)
	if password == "" {
		log.Printf("CreateRoomHandler関数部屋の作成に失敗しました")
		http.Error(w, "部屋の作成に失敗しました", http.StatusInternalServerError)
		return
	}

	// ログに出力して確認
	log.Printf("CreateRoomHandler関数 Room Name: %s, Password: %s", req.RoomName, password)

	resp := map[string]string{
		"password": password,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ルームごとの数字を取得するハンドラー関数
func GetRoomNumbersHandler(w http.ResponseWriter, r *http.Request) {
	// リクエストからルーム名とパスワードを取得
	roomName := r.URL.Query().Get("room_name")
	password := r.URL.Query().Get("password")

	// Room構造体を作成
	room := &Room{
		RoomName: roomName,
		Password: password,
	}

	// 対応するテキストファイルから数字を取得
	fileName := getFileName(room)
	numbers, err := readNumbersFromFile(fileName)
	if err != nil {
		log.Printf("GetRoomNumbersHandler関数ファイルの読み込みに失敗しました: %v", err)
		http.Error(w, "数字の読み取りに失敗しました", http.StatusInternalServerError)
		return
	}

	// クライアントに数字を返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(numbers)
}

// ルームの情報からファイル名を生成する関数
func getFileName(room *Room) string {
	return fmt.Sprintf("%s_%s.txt", room.RoomName, room.Password)
}

// HTTPハンドラー
func handleNumbersRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomName string `json:"room_name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// RoomNameとPasswordが一致するユーザーかどうかを確認
	if !isValidUser(req.RoomName, req.Password) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ファイル名を構成
	fileName := fmt.Sprintf("%s_%s.txt", req.RoomName, req.Password)

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

// ユーザーが正当なRoomNameとPasswordを提供しているかを確認する関数
func isValidUser(roomName, password string) bool {
	// ここで適切な認証メカニズムを実装する
	// 例: ハードコードされた場合
	return roomName == "validRoom" && password == "validPassword"
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

func generateAndWriteNumbersToFiles() {
	for {
		newNumber := generateUniqueNumber()

		// すべてのルームのファイルに数字を書き込む
		for _, room := range roomManager.Rooms {
			fileName := getFileName(room) // ポインタを渡す
			file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("ファイルのオープンに失敗しました: %v", err)
				continue
			}

			if _, err := file.WriteString(fmt.Sprintf("%d\n", newNumber)); err != nil {
				log.Printf("ファイルへの書き込みに失敗しました: %v", err)
			}

			file.Close() // ファイルの使用が終わったら明示的にクローズする
		}
	}
}

// 重複しない数字を生成する関数
func generateUniqueNumber() int {
	newNumber := rand.Intn(75) + 1
	for contains(generatedNumbers, newNumber) {
		newNumber = rand.Intn(75) + 1
	}
	generatedNumbers = append(generatedNumbers, newNumber)
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

// レスポンス用の構造体
type ResponseData struct {
	Numbers []int `json:"numbers"`
}

// Room構造体
type Room struct {
	ID       string
	RoomName string
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

	if ws != nil {
		room.Mutex.Lock()
		defer room.Mutex.Unlock()
		room.Clients[ws] = true
	}
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
func (rm *RoomManager) CreateRoom(roomName string, roomType string) string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	password := generatePassword(PasswordLength)
	room := &Room{
		RoomName: roomName,
		Type:     roomType,
		Password: password,
		Clients:  make(map[*websocket.Conn]bool),
	}

	rm.Rooms[password] = room

	// ルーム名とパスワードを使ってテキストファイルを生成
	fileName := fmt.Sprintf("%s_%s.txt", roomName, password)
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("ファイルの生成に失敗しました: %v", err)
		return ""
	}
	defer file.Close()
	log.Printf("新しいルームが作成されました. Room Name: %s, Password: %s", roomName, password)

	return password
}

// パスワードに基づいてルームを取得する関数
func (rm *RoomManager) GetRoomByPassword(password string) *Room {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	return rm.Rooms[password]
}

// ルームに参加するためのハンドラー関数
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomName string `json:"room_name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JoinRoomHandler関数リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエスト本文が無効です", http.StatusBadRequest)
		return
	}
	// ルーム名をログに出力
	log.Printf("JoinRoomHandler関数 Room Name: %s, Password: %s", req.RoomName, req.Password)

	// 部屋に参加する
	success := roomManager.JoinRoom(req.Password, nil)
	if !success {
		log.Printf("JoinRoomHandler関数部屋に参加できませんでした: %s", req.Password)
		http.Error(w, "部屋に参加できませんでした", http.StatusInternalServerError)
		return
	}

	// ビンゴカードデータを生成
	bingoCard := generateBingoCard()

	// BingoCard型を[][]interface{}型に変換
	convertedCard := make([][]interface{}, 5)
	for i := range bingoCard {
		convertedCard[i] = make([]interface{}, 5)
		for j := range bingoCard[i] {
			if bingoCard[i][j] == 0 {
				convertedCard[i][j] = "FREE"
			} else {
				convertedCard[i][j] = bingoCard[i][j]
			}
		}
	}

	// レスポンスデータを構造体に格納
	response := struct {
		Card     [][]interface{} `json:"card"`
		Interval int             `json:"interval"`
	}{
		Card:     convertedCard,
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
