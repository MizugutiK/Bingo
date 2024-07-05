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
		return true // オリジンのチェックを無効化してすべてのオリジンからの接続を許可する
	},
}

// RoomManager構造体
type RoomManager struct {
	Rooms map[string]*Room // ルームを管理するマップ
	Mutex sync.Mutex       // Roomsへのアクセスを同期するためのミューテックス
}

// Room構造体
type Room struct {
	Password  string                   // ルームのパスワード
	Clients   map[*websocket.Conn]bool // 接続されているクライアントのマップ
	Mutex     sync.Mutex               // Clientsへのアクセスを同期するためのミューテックス
	Interval  int                      // ルーム全体のインターバル値
	Countdown int                      // インターバルの残り時間
	done      chan struct{}            // ゴルーチンの終了シグナル用のチャネル
}

// レスポンス用の構造体
type ResponseData struct {
	Numbers int `json:"numbers"` // JSONレスポンスの構造体
}

// 新しいRoomManagerインスタンスを作成
func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room), // 新しいルームを作成するためのマップ
	}
}

// StartCountdown 関数はルームのカウントダウンを開始します
func (rm *RoomManager) StartCountdown(room *Room) {
	room.done = make(chan struct{})       // 終了シグナル用のチャネルを作成
	ticker := time.NewTicker(time.Second) // 1秒ごとにtickするタイマーを作成

	go func() {
		defer ticker.Stop() // タイマーを停止する
		for {
			select {
			case <-ticker.C:
				room.Mutex.Lock()
				room.Countdown = (room.Countdown - 1 + room.Interval) % room.Interval // インターバルのカウントダウンを計算する
				room.Mutex.Unlock()

				// クライアントに残り時間を送信する処理を追加する（未実装）

			case <-room.done:
				return // ゴルーチンを終了する
			}
		}
	}()
}

// WebSocket接続を処理する関数
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// WebSocket 接続処理
	conn, err := upgrader.Upgrade(w, r, nil) // WebSocketをアップグレードする
	if err != nil {
		log.Fatalf("WebSocket アップグレード エラー: %v", err)
		http.Error(w, "WebSocket アップグレード エラー", http.StatusInternalServerError)
		return
	}
	defer conn.Close() // 関数終了時に接続を閉じる

	// 初回メッセージでルーム名とパスワードを受け取る
	var req struct {
		Password string `json:"password"` // 初回メッセージの構造体
	}
	if err := conn.ReadJSON(&req); err != nil {
		log.Printf("初回メッセージの読み取りエラー: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("初回メッセージの読み取りエラー")) // エラー詳細をクライアントに送信
		return
	}

	// ルームを作成または既存のルームに参加する
	room, exists := roomManager.Rooms[req.Password]
	if !exists {
		// ルームが存在しない場合は新しいルームを作成する
		interval := 60                                   // 例としてインターバル値を設定（必要に応じて変更）
		roomPassword := roomManager.CreateRoom(interval) // 新しいルームを作成する

		// クライアントに新しいルームの情報を送信
		conn.WriteJSON(map[string]interface{}{
			"message":       "新しいルームが作成されました",
			"roomPassword":  roomPassword,
			"interval":      interval,
			"remainingTime": interval, // 初回はインターバル値で設定
		})

		room = roomManager.Rooms[roomPassword] // ルームを更新
	} else {
		// 既存のルームに参加する
		room.Mutex.Lock()
		interval := room.Interval   // インターバルを取得
		countdown := room.Countdown // カウントダウンを取得
		room.Clients[conn] = true   // クライアントにルームを追加
		room.Mutex.Unlock()

		// クライアントにルームの情報を送信
		conn.WriteJSON(map[string]interface{}{
			"message":       "部屋に参加しました",
			"interval":      interval,
			"remainingTime": countdown,
		})
	}

	// クライアントからのメッセージを待機するループ
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("接続が切れました: %v", err)
			room.Mutex.Lock()
			delete(room.Clients, conn) // クライアントを削除
			room.Mutex.Unlock()
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
		return false // パスワードに対応するルームが存在しない場合は参加できない
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	room.Clients[ws] = true // WebSocket接続をルームに追加する
	return true             // 参加成功
}

// ルーム作成関数
func (rm *RoomManager) CreateRoom(interval int) string {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	password := generatePassword(PasswordLength) // ランダムなパスワードを生成
	room := &Room{
		Password: password,                       // パスワードを設定
		Clients:  make(map[*websocket.Conn]bool), // WebSocket接続のマップを初期化
		Interval: interval,                       // インターバルを設定
	}

	rm.Rooms[password] = room // パスワードをキーにしてルームを登録

	log.Printf("新しいルームが作成されました. Password: %s, Interval: %d", password, interval)
	log.Printf("現在のルーム一覧: %v", rm.Rooms) // 現在のルーム一覧をログに出力

	return password // 作成したルームのパスワードを返す
}

// 部屋を作成するハンドラー関数
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Interval int `json:"interval"` // リクエストからのインターバル値
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエストのデコードエラー", http.StatusBadRequest)
		return
	}

	password := roomManager.CreateRoom(req.Interval) // リクエストされたインターバルで新しいルームを作成
	if password == "" {
		log.Println("部屋の作成に失敗しました")
		http.Error(w, "部屋の作成に失敗しました", http.StatusInternalServerError)
		return
	}

	// パスワードに対応するルームを取得
	room := roomManager.GetRoomByPassword(password)
	if room == nil {
		log.Printf("ルームが見つかりませんでした: %s", password)
		http.Error(w, "ルームが見つかりませんでした", http.StatusInternalServerError)
		return
	}

	// レスポンスデータを構築
	resp := map[string]string{
		"password": room.Password, // レスポンスにパスワードを含める
	}

	// レスポンスをJSON形式で返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ルームの数字を一つずつ取得するハンドラー関数
func GetRoomNumbersHandler(w http.ResponseWriter, r *http.Request) {
	password := r.URL.Query().Get("password")
	log.Printf("GetRoomNumbersHandler 関数 リクエストされたパスワード: %s", password)

	// パスワードが提供されていない場合のエラーハンドリング
	if password == "" {
		log.Println("パスワードが提供されていません")
		http.Error(w, "パスワードが提供されていません", http.StatusBadRequest)
		return
	}

	// パスワードに対応するルームを取得
	room := roomManager.GetRoomByPassword(password)
	if room == nil {
		log.Printf("ルームが見つかりませんでした: %s", password)
		http.Error(w, "ルームが見つかりませんでした", http.StatusInternalServerError)
		return
	}

	// ルームの数字を一つずつ取得する
	numbers, err := roomManager.GetNumbersForRoom(password)
	if err != nil {
		log.Printf("数字の取得に失敗しました: %v", err)
		http.Error(w, fmt.Sprintf("数字の取得に失敗しました: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("取得した数字: %v", numbers)

	// サーバーサイドで一つずつ数字を送信する
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	// カウントごとに部分的な配列を生成してクライアントに送信
	for i := 0; i < len(numbers); i++ {
		partNumbers := numbers[:i+1] // i番目までの部分配列を取得
		log.Printf("カウント%d: %v", i+1, partNumbers)

		// JSONエンコードしてクライアントに送信
		if err := encoder.Encode(partNumbers); err != nil {
			log.Printf("JSONエンコードに失敗しました: %v", err)
			http.Error(w, "サーバーエラー", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("\n"))    // 改行を追加して区切る
		w.(http.Flusher).Flush() // フラッシュしてクライアントに送信

		// インターバル待機
		time.Sleep(time.Duration(room.Interval) * time.Second)
	}
}

// ルームの情報からファイル名を生成する関数
func getFileName(room *Room) string {
	return fmt.Sprintf("%s.txt", room.Password) // ルームのパスワードをファイル名に使用
}

// テキストファイルから数字を読み取る関数
func readNumbersFromFile(fileName string) ([]int, error) {
	var numbers []int

	file, err := os.Open(fileName)
	if err != nil {
		return numbers, err // ファイルオープンエラーを返す
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		num, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return numbers, err // 数字の読み取りエラーを返す
		}
		numbers = append(numbers, num) // 数字をスライスに追加
	}

	return numbers, nil // 読み取った数字のスライスを返す
}

// ルームごとに数字を生成してファイルに書き込むループ
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
			// log.Printf("生成されたファイル名: %s", fileName) // デバッグ用にファイル名をログ出力

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

		// 少し待機してから再試行する
		time.Sleep(time.Second * 1)
	}
}

// GetNumbersForRoomメソッドを定義
func (rm *RoomManager) GetNumbersForRoom(password string) ([]int, error) {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	room, exists := rm.Rooms[password]
	if !exists {
		log.Printf("GetNumbersForRoom: ルームが見つかりません: %s", password) // ルームが存在しない場合のエラーログ
		return nil, fmt.Errorf("ルームが見つかりません: %s", password)
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// ファイル名を取得
	fileName := getFileName(room)

	// テキストファイルから数字を読み取る
	numbers, err := readNumbersFromFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("数字の読み取りに失敗しました: %v", err)
	}

	return numbers, nil // 読み取った数字のスライスを返す
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

// ルーム管理のためのインスタンス
var roomManager = NewRoomManager()

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

	// ルームごとの数字取得エンドポイント
	http.HandleFunc("/get-room-numbers", GetRoomNumbersHandler)

	// サーバーの起動
	log.Println("Listening on :8080...")
	go generateAndWriteNumbersToFiles() // 数字生成のゴルーチンを起動
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var generatedNumbers []int // 重複をチェックするためのスライス

// ルームに関する定数と構造体
const (
	PasswordLength = 6 // ルームのパスワードの長さ
)

// パスワードに基づいてルームを取得する関数
func (rm *RoomManager) GetRoomByPassword(password string) *Room {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	return rm.Rooms[password] // パスワードに対応するルームを返す
}

// ルームに参加するためのハンドラー関数
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "無効なHTTPメソッド", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"` // JSONからのパスワードリクエスト
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエスト本文が無効です", http.StatusBadRequest)
		return
	}

	// ルームに参加
	success := roomManager.JoinRoom(req.Password, nil) // WebSocket接続は未実装
	if !success {
		log.Printf("部屋に参加できませんでした: %s", req.Password)
		http.Error(w, "部屋に参加できませんでした", http.StatusUnauthorized)
		return
	}

	log.Printf("JoinRoomHandler: 部屋に参加しました: %s", req.Password) // 部屋参加成功時のログ

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "部屋に参加しました"})
}

// ビンゴカードを生成するハンドラー関数
func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	bingoCard := generateBingoCard() // ビンゴカードを生成
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bingoCard) // ビンゴカードをJSONで返す
}

// ビンゴチェックを行うハンドラー関数
func CheckBingoHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Card   BingoCard  `json:"card"`   // ビンゴカード
		Marked [5][5]bool `json:"marked"` // マークされたセルの状態
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("リクエストのデコードエラー: %v", err)
		http.Error(w, "リクエスト本文が無効です", http.StatusBadRequest)
		return
	}

	isBingo := checkBingo(req.Card, req.Marked) // ビンゴをチェック
	resp := map[string]bool{"bingo": isBingo}   // レスポンスを準備
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) // ビンゴの結果をJSONで返す
}

// 生成された数字のリストをリセットするハンドラー関数
func ResetGeneratedNumbersHandler(w http.ResponseWriter, r *http.Request) {
	generatedNumbers = []int{} // 生成された数字をリセット
	response := map[string]string{"message": "生成された番号はリセットされました"}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "JSON 応答の生成に失敗しました", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse) // レスポンスをJSONで返す
}

// BingoCard型の定義
type BingoCard [5][5]int // ビンゴカードの型定義

// ビンゴカードを生成する関数
func generateBingoCard() BingoCard {
	rand.Seed(time.Now().UnixNano()) // ランダムシードの初期化

	var card BingoCard
	usedNumbers := make(map[int]bool) // 使用済みの数字を管理するマップ

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			num := rand.Intn(75) + 1 // 1から75までのランダムな数字を生成
			for usedNumbers[num] {
				num = rand.Intn(75) + 1 // 既に使用されている場合は再生成
			}
			usedNumbers[num] = true // 使用済みマップに追加
			card[i][j] = num        // カードに数字をセット
		}
	}

	card[2][2] = 0 // FREE space (中央のマスを0でセット)

	return card // 生成されたビンゴカードを返す
}

// ビンゴをチェックする関数
func checkBingo(card BingoCard, marked [5][5]bool) bool {
	// 横方向のチェック
	for i := 0; i < 5; i++ {
		if marked[i][0] && marked[i][1] && marked[i][2] && marked[i][3] && marked[i][4] {
			return true // 横一列が全てマークされている場合、ビンゴ
		}
	}

	// 縦方向のチェック
	for j := 0; j < 5; j++ {
		if marked[0][j] && marked[1][j] && marked[2][j] && marked[3][j] && marked[4][j] {
			return true // 縦一列が全てマークされている場合、ビンゴ
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
		return true // 左上から右下の斜めが全てマークされている場合、ビンゴ
	}

	// 斜め方向のチェック（右上から左下）
	diagonal2 := true
	for i := 0; i < 5; i++ {
		if !marked[i][4-i] {
			diagonal2 = false
			break
		}
	}
	return diagonal2 // 右上から左下の斜めが全てマークされている場合、ビンゴ
}
