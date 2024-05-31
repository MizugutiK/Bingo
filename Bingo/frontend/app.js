// 必要な要素を取得
const bingoCard = document.getElementById('bingo-card');
// 次の数字を表示する要素を取得
const numberDiv = document.getElementById('number');
// カウントダウンを表示する要素を取得
const countdownDiv = document.getElementById('countdown');
// 生成された数字のログを表示する要素を取得
const logDiv = document.getElementById('log');

// ゲーム開始
const newgameBoton = document.getElementById('new-game');
// リセットボタンを取得
const resetButton = document.getElementById('reset-game'); 

// カウントダウンのインターバルを管理する変数
let countdownInterval;
let countdownTime = 60; // カウントダウンの初期値を設定（秒）

// 生成された数字のリスト
let generatedNumbers = [];

// ビンゴの数字が生成されたときに再生するSEの音声ファイルのパス
const audioPath = 'chime.mp3';

// WebSocketを作成し、サーバーとの接続を確立
const ws = new WebSocket('ws://localhost:8080/ws');

// WebSocket接続が確立されたときの処理
ws.onopen = function(event) {
    console.log('WebSocket connection established.');
};

// WebSocketからメッセージを受信したときの処理
ws.onmessage = function(event) {
    // console.log('Received message:', event.data);
    handleNewNumber(event.data);
};

// WebSocketエラーが発生したときの処理
ws.onerror = function(error) {
     console.error('WebSocket error:', error);
};

// 新しいゲームを開始するボタンのクリックイベントリスナーを追加
document.getElementById('new-game').addEventListener('click', () => {
    // 新しいゲームをサーバーにリクエスト
    fetch('/new-game')
        .then(response => response.json())
        .then(data => {
            // ビンゴカードをレンダリング
            renderBingoCard(data);
            // カウントダウンを開始
            startCountdown();
            // ログをクリア
            logDiv.innerHTML = '';

            // ゲームが始まった瞬間、0を生成された数字のリストに追加してクリック可能にする
            generatedNumbers = [0];
            enableClickableCells();

            // リセットボタンを非表示に
            resetButton.style.display = 'none';
            newgameBoton.style.display = "none";

        })
        .catch(error => console.error('Error:', error));
});

// カウントダウンを開始する関数
function startCountdown() {
    // カウントダウンインターバルをクリア
    clearInterval(countdownInterval);
    countdownTime = 60; // カウントダウン時間を60秒に設定
    updateCountdown();
    // カウントダウンを1秒ごとに実行
    countdownInterval = setInterval(() => {
        countdownTime--;
        updateCountdown();
        // カウントダウンが終了したらカウントダウンを停止
        if (countdownTime <= 0) {
            clearInterval(countdownInterval);
            countdownDiv.textContent = '';
        }
    }, 1000);
}

// カウントダウンを更新する関数
function updateCountdown() {
    // 分単位と秒単位を計算
    const seconds = countdownTime % 60;
    // カウントダウン要素に表示
    countdownDiv.textContent = `${seconds < 10 ? '0' : ''}${seconds}`;
}

// セルがクリック可能かどうかを判断する関数
function isClickableCell(cellValue) {
    return !generatedNumbers.includes(cellValue); // 生成された数字のリストと比較して、セルの数字が含まれていない場合はクリック可能とする
}

// 新しい数字を取得したときの処理
function handleNewNumber(data) {
    const numbers = JSON.parse(data);
    const latestNumber = numbers[numbers.length - 1];

    // 新しい数字を generatedNumbers リストに追加
    generatedNumbers.push(latestNumber);
    // console.log('Generated numbers:', generatedNumbers);

    // ビンゴカードのセルをクリック可能にする
    enableClickableCells(latestNumber);

    // 数字を表示
    numberDiv.textContent = `Newナンバー: ${latestNumber}`;

    // ログに追加
    logDiv.innerHTML = ''; // ログをクリアして再描画
    numbers.forEach(number => {
        const logItem = document.createElement('div');
        logItem.textContent = `ログ:  ${number}`;
        logDiv.appendChild(logItem);
    });

    // カウントダウンを再スタート
    startCountdown();
    // SEを再生
    playAudio(audioPath);

    // すべての番号が生成されたかどうかを確認
    if (generatedNumbers.length >= 76) {
        // リセットボタンを表示
        resetButton.style.display = 'block';
    }
}

// ビンゴカードのセルをクリック可能にする関数
function enableClickableCells() {
    // ビンゴカードのすべてのセルを取得
    const cells = document.querySelectorAll('.cell');
    // 生成された数字のリストをもとに、クリック可能なセルを設定
    cells.forEach(cell => {
        // セルに表示されている数字を取得（FREEセルの場合は'FREE'）
        const cellNumber = cell.textContent === 'FREE' ? 0 : parseInt(cell.textContent);
        // ビンゴカードの数字が生成された数字のリストに含まれている場合、またはセルが0である場合、クリック可能にする
        if (generatedNumbers.includes(cellNumber) || cellNumber === 0) {
            // 既にクリック可能になっているかどうかをチェック
            if (!cell.classList.contains('clickable')) {
                cell.classList.add('clickable'); // クリック可能にするためのクラスを追加
                cell.addEventListener('click', cellClickHandler); // クリックイベントを追加
            }
        } else {
            // 数字が一致しない場合、クリック可能なクラスとイベントを削除
            cell.classList.remove('clickable');
            cell.removeEventListener('click', cellClickHandler);
        }
    });

    // 中央のセルを常にクリック可能にする
    const centerCell = document.querySelector('[data-row-index="2"][data-cell-index="2"]');
    centerCell.classList.add('clickable');
    centerCell.addEventListener('click', cellClickHandler);
}

// SEを再生する関数
function playAudio(audioPath) {
    const audio = new Audio(audioPath);
    audio.play();
}

// ビンゴカードをレンダリングする関数
function renderBingoCard(data) {
    bingoCard.innerHTML = ''; // ビンゴカードを初期化
    // マーク状態を保持する配列を初期化
    window.marked = Array.from({ length: 5 }, () => Array(5).fill(false)); 
    data.forEach((row, i) => {
        // console.log(`Row ${i + 1}: Number of cells: ${row.length}`);
        // ビンゴカードの各行ごとにセルを生成
        for (let j = 0; j < 5; j++) {
            // セル要素を作成
            const cellDiv = document.createElement('div');
            cellDiv.className = 'cell';
            cellDiv.dataset.rowIndex = i; // データ属性に行インデックスを追加
            cellDiv.dataset.cellIndex = j; // データ属性に列インデックスを追加
            // セルに数字を表示（FREEセルは0を表示）
            if (j < row.length) {
                cellDiv.textContent = row[j] !== 0 ? row[j] : 'FREE';
            } else {
                cellDiv.textContent = ''; // セルが存在しない場合は空白にする
            }
            // セルがクリック可能かどうかをチェック
            if (j < row.length && row[j] !== 0 && isClickableCell(row[j])) {
                cellDiv.classList.add('clickable');
                cellDiv.addEventListener('click', cellClickHandler);
            }
            // 中央のセルであればクリック可能にする
            if (i === 2 && j === 2) {
                cellDiv.classList.add('clickable');
                cellDiv.addEventListener('click', cellClickHandler);
            }
            // セルをビンゴカードに追加
            bingoCard.appendChild(cellDiv);

            // マークされたカードの配列を生成
            if (!window.marked[i]) {
                window.marked[i] = [];
            }
        }
    });
    enableClickableCells(); // 新しい数字が生成された後、クリック可能なセルを再設定
}

// セルがクリックされたときの処理
function cellClickHandler() {
    // クリックされたセルがマークされた状態かどうかを切り替える
    const rowIndex = parseInt(this.dataset.rowIndex); // データ属性から行インデックスを取得
    const cellIndex = parseInt(this.dataset.cellIndex); // データ属性から列インデックスを取得
    window.marked[rowIndex][cellIndex] = !window.marked[rowIndex][cellIndex];
    this.classList.toggle('marked');
    // console.log(`Cell clicked: row ${rowIndex}, column ${cellIndex}, marked: ${window.marked[rowIndex][cellIndex]}`);

    // ビンゴをチェック
    checkBingo();
}

// SEを再生する関数
function playAudio(audioPath) {
    const audio = new Audio(audioPath);
    audio.play();
}

// WebSocketエラー発生時の処理
ws.onerror = function(error) {
    console.error('WebSocket error:', error);
};

// ビンゴをチェックする関数
function checkBingo() {
    // console.log('Checking bingo...');
    console.log('Marked card:', window.marked);

    // ビンゴをチェックするリクエストをサーバーに送信
    fetch('/check-bingo', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ marked: window.marked }) // bingoCard ではなく marked を送信
    })
    .then(response => {
        // console.log('Response received:', response);

        // レスポンスをJSONとして解析
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(data => {
        // console.log('Bingo check result:', data);

        // ビンゴが達成された場合はアラートを表示
        if (data.bingo) {
            alert('ビンゴです！');
            newgameBoton.style.display = "block";
        }
    })
    .catch(error => console.error('Error:', error.message));
}

// リセットボタンのクリックイベントリスナーを追加
resetButton.addEventListener('click', () => {
    // 生成された数字のリストをリセット
    generatedNumbers = [];
    // ビンゴカードをクリア
    bingoCard.innerHTML = '';
    // 数字表示をクリア
    numberDiv.textContent = '';
    // ログをクリア
    logDiv.innerHTML = '';
    // カウントダウンをリセット
    clearInterval(countdownInterval);
    countdownDiv.textContent = '';

    // リセットボタンを非表示に
    resetButton.style.display = 'none';
});
