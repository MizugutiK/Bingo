// app.js

// ビンゴカード要素を取得
const bingoCard = document.getElementById('bingo-card');
// 次の数字を表示する要素を取得
const numberDiv = document.getElementById('number');
// カウントダウンを表示する要素を取得
const countdownDiv = document.getElementById('countdown');

// カウントダウンのインターバルを管理する変数
let countdownInterval;
// カウントダウンの初期値を設定（秒）
let countdownTime = 60;

// ビンゴの数字が生成されたときに再生するSEの音声ファイルのパス
const audioPath = 'chime.mp3';

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
        })
        .catch(error => console.error('Error:', error));
});

// カウントダウンを開始する関数
function startCountdown() {
    // カウントダウンインターバルをクリア
    clearInterval(countdownInterval);
    // カウントダウン時間を60秒に設定
    countdownTime = 60;
    // カウントダウンを更新
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
    // const minutes = Math.floor(countdownTime / 60);
    const seconds = countdownTime % 60;
    // カウントダウン要素に表示
    countdownDiv.textContent = `${seconds < 10 ? '0' : ''}${seconds}`;
}

// ビンゴカードをレンダリングする関数
function renderBingoCard(data) {
    // ビンゴカードを初期化
    bingoCard.innerHTML = '';
    // ビンゴカードのデータを元にセルを作成
    data.forEach((row, i) => {
        row.forEach((cell, j) => {
            // セル要素を作成
            const cellDiv = document.createElement('div');
            cellDiv.className = 'cell';
            // セルに数字を表示（FREEセルは0を表示）
            cellDiv.textContent = cell !== 0 ? cell : 'FREE';
            // セルがクリックされた時のイベントリスナーを追加
            cellDiv.addEventListener('click', () => {
                // クリックされたセルがマークされた状態かどうかを切り替える
                cellDiv.classList.toggle('marked');
                // マークされたセルの状態を記録
                window.marked[i][j] = !window.marked[i][j];
                // ビンゴをチェック
                checkBingo();
            });
            // セルをビンゴカードに追加
            bingoCard.appendChild(cellDiv);
        });
    });
}

// WebSocketを作成し、サーバーとの接続を確立
const ws = new WebSocket('ws://localhost:8080/ws');

// サーバーからメッセージを受信したときの処理
ws.onmessage = function(event) {
    // 受信したデータをパースして次の数字を取得し、表示
    const number = JSON.parse(event.data);
    numberDiv.textContent = `次の番号: ${number}`;
    // カウントダウンを再スタート
    startCountdown();
      // SEを再生
      playAudio(audioPath);
};

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
    // ビンゴをチェックするリクエストをサーバーに送信
    fetch('/check-bingo', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ card: window.bingoCard, marked: window.marked })
    })
    .then(response => response.json())
    .then(data => {
        // ビンゴが達成された場合はアラートを表示
        if (data.bingo) {
            alert('ビンゴです！');
        }
    })
    .catch(error => console.error('Error:', error));
}
