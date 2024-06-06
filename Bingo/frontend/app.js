let createRoomButton; // createRoomButtonを外部スコープで定義
document.addEventListener("DOMContentLoaded", function() {
    const cells = document.querySelectorAll(".cell");

    // すべてのセルのフォントサイズを調整
    cells.forEach(function(cell) {
        adjustFontSize(cell);
    });

    // ウィンドウサイズが変更されたときにフォントサイズを再調整
    window.addEventListener("resize", function() {
        cells.forEach(function(cell) {
            adjustFontSize(cell);
        });
    });

let ws;    
// WebSocketの設定
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsHost = `${wsProtocol}//localhost:8080/ws`;

// createRoomButtonを定義
 createRoomButton = document.getElementById('create-room'); 
// create-room ボタンのクリックイベントリスナーを追加
createRoomButton.addEventListener('click', () => {

     

    // ルームを作成するリクエストをサーバーに送信
    fetch('/create-room', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ host: 'Your Host Name', room_type: 'public' }) // ホスト名とルームタイプを指定（適切な値に変更）
    })
    .then(response => response.json())
    .then(data => {
        if (data.password) {
            alert(`Generated Password: ${data.password}`);
            console.log('Password:', data.password);
        } else {
            alert('Failed to generate room password');
        }
    })
    .catch(error => {
        console.error('Error:', error.message);
    });

});

 // ルーム参加ボタンと入力フィールドを取得
 const joinRoomButton = document.getElementById('join-room');
// ルーム参加ボタンにクリックイベントリスナーを追加
joinRoomButton.addEventListener('click', () => {
    const password = document.getElementById('room-password').value;
    console.log('Password:', password); // パスワードをログに出力

    fetch('/join-room', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ password: password })
    })
    .then(response => response.json())
    .then(data => {
        console.log('Response data:', data); // データをログに出力
        if (data.room_id) {
            alert(`Joined room ${data.room_id}`);
            // WebSocket接続を確立
            ws = new WebSocket(`${wsHost}?room_id=${data.room_id}`);

            ws.onopen = () => {
                console.log('WebSocket connection established for room:', data.room_id);
            };

            ws.onmessage = event => {
                handleNewNumber(event.data);
            };

            ws.onerror = error => {
                console.error('WebSocket error:', error);
            };

            ws.onclose = event => {
                console.log('WebSocket connection closed:', event);
            };
        } else {
            alert('Failed to join room');
        }
    })
    .catch(error => {
        console.error('Error:', error.message);
    });

});

    // 必要な要素を取得
    const bingoCard = document.getElementById('bingo-card');
    const numberDiv = document.getElementById('number');
    const countdownDiv = document.getElementById('countdown');
    const logDiv = document.getElementById('log');
    const newgameBoton = document.getElementById('new-game');
    const resetButton = document.getElementById('reset-game'); 

    let countdownInterval;
    let countdownTime = 60;
    let generatedNumbers = [];
    const audioPath = 'chime.mp3';

// WebSocketのイベントリスナーを定義
function initializeWebSocket() {
    ws = new WebSocket(wsHost);

    ws.onopen = function(event) {
        console.log('WebSocket接続が確立された.');
    };

    ws.onmessage = function(event) {
        const generatedNumbersFromServer = JSON.parse(event.data);
        generatedNumbers = generatedNumbersFromServer;
        enableClickableCells();
        handleNewNumber(event.data);
    };

    ws.onerror = function(error) {
        console.error(' WebSocketエラー:', error);
    };

    ws.onclose = function(event) {
        console.log('WebSocket接続が閉じた:', event);
    };
}

initializeWebSocket();

    // 新しいゲームを開始するボタンのクリックイベントリスナーを追加
    newgameBoton.addEventListener('click', () => {
        fetch('/new-game')
            .then(response => response.json())
            .then(data => {
                renderBingoCard(data);
                startCountdown();
                generatedNumbers = data.generatedNumbers;
                enableClickableCells();
                resetButton.style.display = 'none';
                newgameBoton.style.display = "none";
                if (createRoomButton) {
                    createRoomButton.style.display = "none"; // createRoomButtonを参照
                }
            })
            .catch(error => console.error('Error:', error));
    });

    // カウントダウンを開始する関数
    function startCountdown() {
        clearInterval(countdownInterval);
        countdownTime = 60;
        updateCountdown();
        countdownInterval = setInterval(() => {
            countdownTime--;
            updateCountdown();
            if (countdownTime <= 0) {
                clearInterval(countdownInterval);
                countdownDiv.textContent = '';
            }
        }, 1000);
    }

    // カウントダウンを更新する関数
    function updateCountdown() {
        const seconds = countdownTime % 60;
        countdownDiv.textContent = `${seconds < 10 ? '0' : ''}${seconds}`;
    }

    // セルがクリック可能かどうかを判断する関数
    function isClickableCell(cellValue) {
        return !generatedNumbers.includes(cellValue); 
    }

    // 新しい数字を取得したときの処理
    function handleNewNumber(data) {
        const numbers = JSON.parse(data);
        const latestNumber = numbers[numbers.length - 1];

        generatedNumbers.push(latestNumber);
        enableClickableCells(latestNumber);

        numberDiv.textContent = `Newナンバー: ${latestNumber}`;

        logDiv.innerHTML = '';
        numbers.forEach(number => {
            const logItem = document.createElement('div');
            logItem.textContent = `ログ:  ${number}`;
            logDiv.appendChild(logItem);
        });

        startCountdown();
        playAudio(audioPath);

        if (generatedNumbers.length >= 76) {
            resetButton.style.display = 'block';
        }
    }

    // ビンゴカードのセルをクリック可能にする関数
    function enableClickableCells() {
        const cells = document.querySelectorAll('.cell');
        if (!cells || cells.length === 0) {
            return; // セルが存在しない場合は処理を中止
        }
        cells.forEach(cell => {
            const cellNumber = cell.textContent === 'FREE' ? 0 : parseInt(cell.textContent);
            if (Array.isArray(generatedNumbers) && (generatedNumbers.includes(cellNumber) || cellNumber === 0)) {
                if (!cell.classList.contains('clickable')) {
                    cell.classList.add('clickable');
                    cell.addEventListener('click', cellClickHandler);
                }
            } else {
                cell.classList.remove('clickable');
                cell.removeEventListener('click', cellClickHandler);
            }
        });

        const centerCell = document.querySelector('[data-row-index="2"][data-cell-index="2"]');
        centerCell.classList.add('clickable');
        centerCell.addEventListener('click', cellClickHandler);
    }

    function cellClickHandler() {
        const rowIndex = parseInt(this.dataset.rowIndex);
        const cellIndex = parseInt(this.dataset.cellIndex);
        window.marked[rowIndex][cellIndex] = !window.marked[rowIndex][cellIndex];
        this.classList.toggle('marked');
        checkBingo();
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
                    cellDiv.textContent = row[j] !== 0 ? row[j] : '☆';
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
                adjustFontSize(cellDiv); // 追加されたセルのフォントサイズを調整
            }
        });
        enableClickableCells(); // 新しい数字が生成された後、クリック可能なセルを再設定
    }

    // WebSocketエラー発生時の処理
    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
    };

    // ビンゴをチェックする関数
    function checkBingo() {
        // console.log('Checking bingo...');
        // console.log('Marked card:', window.marked);

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
        fetch('/reset-generated-numbers')
        .then(response => {
            if (response.ok) {
                // 生成された数字のリストをリセット
                generatedNumbers = [];
                // 数字表示をクリア
                numberDiv.textContent = '';
                // ログをクリア
                logDiv.innerHTML = '';
                // カウントダウンをリセット
                clearInterval(countdownInterval);
                countdownDiv.textContent = '';

                // リセットボタンを非表示に
                resetButton.style.display = 'none';
                newgameBoton.style.display = "block";
            } else {
                console.error('Failed to reset generated numbers');
            }
        })
        .catch(error => console.error('Error:', error));
    });

    function adjustFontSize(cell) {
        var cellSize = Math.min(cell.offsetWidth, cell.offsetHeight);
        var fontSize = cellSize * 0.35; // セルのサイズに基づいたフォントサイズ
        cell.style.fontSize = fontSize + "px";
    }
});
