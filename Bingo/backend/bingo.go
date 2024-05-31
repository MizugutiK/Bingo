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
	// log.Println("Checking Bingo...")

	// 横方向のチェック
	for i := 0; i < 5; i++ {
		if marked[i][0] && marked[i][1] && marked[i][2] && marked[i][3] && marked[i][4] {
			// log.Printf("Bingo found in row %d\n", i)
			return true
		}
	}

	// 縦方向のチェック
	for j := 0; j < 5; j++ {
		if marked[0][j] && marked[1][j] && marked[2][j] && marked[3][j] && marked[4][j] {
			// log.Printf("Bingo found in column %d\n", j)
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
		// log.Println("Bingo found in left-to-right diagonal")
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
	if diagonal2 {
		// log.Println("Bingo found in right-to-left diagonal")
		return true
	}

	// log.Println("No Bingo found")
	return false
}
