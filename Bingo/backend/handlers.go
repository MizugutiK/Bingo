package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// NewGameHandlerは新しいビンゴゲームを開始し、ビンゴカードを生成してクライアントに返します
func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	// // 生成された数字のリストをリセット
	// resetGeneratedNumbers()

	// ビンゴカードを生成
	bingoCard := generateBingoCard()

	// JSON形式でレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bingoCard)
}

// CheckBingoHandlerはビンゴが達成されたかどうかをチェックします
func CheckBingoHandler(w http.ResponseWriter, r *http.Request) {
	// log.Println("CheckBingoHandler called")

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

	// log.Printf("Card: %+v\n", req.Card)
	// log.Printf("Marked: %+v\n", req.Marked)

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

// // 生成された数字のリストをリセットする関数
// func resetGeneratedNumbers() {
// 	generatedNumbers = []int{}
// }
