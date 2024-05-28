package main

import (
	"encoding/json"
	"net/http"
)

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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// ビンゴが達成されたかをチェック
	isBingo := checkBingo(req.Card, req.Marked)

	// レスポンスをJSON形式で返す
	json.NewEncoder(w).Encode(map[string]bool{"bingo": isBingo})
}
