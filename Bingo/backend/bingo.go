package main

import (
	"math/rand"
	"time"
)

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

	// 斜め方向のチェック
	if marked[0][0] && marked[1][1] && marked[2][2] && marked[3][3] && marked[4][4] {
		return true
	}
	if marked[0][4] && marked[1][3] && marked[2][2] && marked[3][1] && marked[4][0] {
		return true
	}

	return false
}
