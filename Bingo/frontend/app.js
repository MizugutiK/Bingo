let ws;
let generateNumbersEnabled = false; // 初期状態はfalse

// DOMContentLoaded イベントでページが読み込まれた後に実行される
document.addEventListener("DOMContentLoaded", function() {
    initializeWebSocket();
    adjustAllCellFonts();
    setupEventListeners();
});

// WebSocketの設定
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsHost = `${wsProtocol}//localhost:8080/ws`;

// WebSocketのイベントリスナーを定義
function initializeWebSocket() {
    ws = new WebSocket(wsHost);

    ws.onopen = function(event) {
        console.log('WebSocket接続が確立された.');
    };

    ws.onmessage = function(event) {
        if (generateNumbersEnabled) {
            const generatedNumbersFromServer = JSON.parse(event.data);
            generatedNumbers = generatedNumbersFromServer;
            enableClickableCells();
            handleNewNumber(event.data);
        }
    };

    ws.onerror = function(error) {
        console.error('WebSocketエラー:', error);
    };

    ws.onclose = function(event) {
        console.log('WebSocket接続が閉じた:', event);
        setTimeout(initializeWebSocket, 1000); // 1秒後に再接続
    };
}

// 必要な要素を取得
const bingoCard = document.getElementById('bingo-card');
const numberDiv = document.getElementById('number');
const countdownDiv = document.getElementById('countdown');
const logDiv = document.getElementById('log');
const roomNameDisplay = document.getElementById('roomNameDisplay');

const resetButton = document.getElementById('reset-game');
const joinRoomButton = document.getElementById('join-room');
const roomNameInput = document.getElementById('roomname');
const createRoomButton = document.getElementById('create-room');
const roomTypeSelect = document.getElementById('room-type');
const setIntervalBtn = document.getElementById('set-interval-btn');
const intervalInput = document.getElementById('interval');
// UI周りの表示非表示用の宣言
const elementsToHide = document.querySelectorAll
('#interval, #set-interval-btn, #CreateRoom, #join-room-container,#reset-game');

// ビンゴカードを非表示にする
document.querySelector('.row.mt-2').style.display = 'none';

function setupEventListeners() {
    resetButton.addEventListener('click', resetGame);
    joinRoomButton.addEventListener('click', joinRoom);
    createRoomButton.addEventListener('click', createRoom);
    setIntervalBtn.addEventListener('click', handleSetIntervalBtnClick);
    window.addEventListener("resize", adjustAllCellFonts);
}

// リセットボタンのクリックイベントリスナーを追加
function resetGame() {
    clearInterval(countdownInterval);
    countdownDiv.textContent = '';
    console.log('番号リセット');
    fetch('/reset-generated-numbers')
        .then(handleResponse)
        .then(() => {
            generatedNumbers = [];
            numberDiv.textContent = '';
            logDiv.innerHTML = '';
            countdownDiv.textContent = '';
            resetButton.style.display = 'none';
            console.log('リセットできた');
        })
        .catch(handleError);
}


// ルームに参加する関数
function joinRoom() {
    const roomName = roomNameInput.value.trim();
    const password = document.getElementById('room-password').value;
    console.log('Password:', password); // パスワードをログに出力

    // ルーム名をブラウザ上に表示
    roomNameDisplay.textContent = `ルーム名: ${roomName}`;

    fetch('/join-room', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ password: password })
    })
    .then(handleJoinRoomResponse)
    .catch(handleJoinRoomError);
}

// WebSocket接続を作成する関数
function createRoom() {
    const roomName = roomNameInput.value.trim(); // ルーム名を取得してトリム
    const roomType = roomTypeSelect.value; // ユーザーが選択したルームタイプを取得

    if (roomName !== '') {
        fetch('/create-room', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ host: 'Your Host Name', room_type: roomType }) // ホスト名と選択したルームタイプを送信
        })
        .then(response => response.json())
        .then(data => {
            if (data.password) {
                // パブリックルームまたはプライベートルームかをログに出力
                console.log(`Generated Password for ${roomType} room: ${data.password}`);
                alert(`Generated Password for ${roomType} room: ${data.password}`);
                console.log('Password:', data.password);
            } else {
                alert('Failed to generate room password');
            }
        })
        .catch(handleError);
    } else {
        // ルーム名が空の場合はアラートを表示するなど、適切な処理を行う
        alert('ルーム名を入力してください。');
    }
}

// インターバル設定ボタンのクリックイベントリスナー
function handleSetIntervalBtnClick() {
    // インターバル設定ボタンがクリックされたら、特定の要素を非表示にする
    elementsToHide.forEach(element => {
        element.style.display = 'none';
    });
    // ビンゴカードを表示する
    document.querySelector('.row.mt-2').style.display = 'block';

    fetch('/new-game')
        .then(response => response.json())
        .then(data => {
            console.log('Data received from /new-game:', data); // デバッグ用
            renderNewGame(data);
            const interval = data.interval !== undefined ? data.interval : 1; // デフォルト値を設定
            startCountdown(interval); // カウントダウンを開始
        })
        .catch(handleError);
    const newInterval = parseInt(intervalInput.value); // 新しいインターバル値を取得し、整数に変換
    if (!isNaN(newInterval) && newInterval > 0) { // 正しい数値かどうかを確認
        fetch('/set-interval', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ interval: newInterval }) // 正しい形式のJSONデータを送信
            })
            .then(response => response.text())
            .then(text => {
                // 受け取ったレスポンスがJSONかどうかを確認
                try {
                    const data = JSON.parse(text);
                    console.log('Interval updated successfully', data);
                    startCountdown(newInterval); // インターバルが設定されたらカウントダウンを開始
                    generateNumbersEnabled = true; // ボタンがクリックされたら生成を有効化
                    resetGame(); // resetGame 関数を呼び出す
                } catch (error) {
                    if (text === 'Interval has been set') {
                        console.log('数字生成間隔更新');
                        startCountdown(newInterval); // インターバルが設定されたらカウントダウンを開始
                        generateNumbersEnabled = true; // ボタンがクリックされたら生成を有効化
                        resetGame(); // resetGame 関数を呼び出す
                    } else {
                        throw new Error(`Invalid JSON response: ${text}`);
                    }
                }
            })
            .catch(handleError);
    } else {
        console.error('Invalid interval value:', intervalInput.value);
    }
}


// 共通のエラーハンドラー
function handleError(error) {
    console.error('Error:', error.message);
    alert(`Error: ${error.message}`);
}


let countdownInterval;
let generatedNumbers = [];
const audioPath = 'chime.mp3';

// セルがクリック可能かどうかを判断する関数
function isClickableCell(cellValue) {
    return !generatedNumbers.includes(cellValue);
}

// 新しい数字を取得したときの処理
function handleNewNumber(data) {
    if (!data) {
        console.error('Empty message received from WebSocket.');
        return;
    }

    const numbers = JSON.parse(data);
    if (!Array.isArray(numbers)) {
        console.error('Invalid message format received from WebSocket.');
        return;
    }

    const latestNumber = numbers[numbers.length - 1];

    // generatedNumbersが配列でない場合は、新しい配列として初期化する
    if (!Array.isArray(generatedNumbers)) {
        generatedNumbers = [];
    }

    generatedNumbers.push(latestNumber);

    enableClickableCells();
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

// ルーム参加リクエストのレスポンスを処理する関数
function handleJoinRoomResponse(response) {
    console.log('Response status:', response.status);
    console.log('Response headers:', response.headers);
    return response.text().then(text => {
        console.log('Raw response text:', text); // レスポンスの生データをログに出力
        if (!response.ok) {
            try {
                const errorJson = JSON.parse(text);
                throw new Error(errorJson.error || 'Failed to join room');
            } catch (e) {
                throw new Error(text || 'Failed to join room');
            }
        }
        if (text.trim() === '') {
            throw new Error('Empty response');
        }
        try {
            return JSON.parse(text);
        } catch (e) {
            throw new Error('Invalid JSON response');
        }
    }).then(handleJoinRoomData);
}

// ルーム参加時のエラーを処理する関数
function handleJoinRoomError(error) {
    console.error('Error:', error.message);
    alert(`Error: ${error.message}`);
}

// ルーム参加時のデータを処理する関数
function handleJoinRoomData(data) {
    console.log('Response data:', data);
    if (data.room_id) {
        alert(`Joined room ${data.room_id}`);
        ws = new WebSocket(`${wsHost}?room_id=${data.room_id}`);

        ws.onopen = () => {
            console.log('WebSocket connection established for room:', data.room_id);
        };

        ws.onmessage = event => {
            console.log('Received WebSocket message:', event.data);
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
}

// 新しいゲームのデータをレンダリングする関数
function renderNewGame(data) {
    renderBingoCard(data);
    startCountdown(data.interval); 
    generatedNumbers = data.generatedNumbers;
    enableClickableCells();

}

// カウントダウンを開始する関数
function startCountdown(newInterval) {
    clearInterval(countdownInterval);
    let countdownTime = newInterval || parseInt(intervalInput.value); // 新しいインターバル値またはinputの値を使用
    updateCountdown(countdownTime);
    countdownInterval = setInterval(() => {
        countdownTime--;
        updateCountdown(countdownTime);
        if (countdownTime <= 0) {
            clearInterval(countdownInterval);
            countdownDiv.textContent = '';
        }
    }, 1000);
}

// カウントダウンを更新する関数
function updateCountdown(time) {
    if (!isNaN(time)) {
        const seconds = time % 60;
        countdownDiv.textContent = `${seconds < 10 ? '0' : ''}${seconds}`;
    } else {
        countdownDiv.textContent = '00'; 
    }
}

// セルがクリック可能になるかどうかを判断する関数
function enableClickableCells() {
    const cells = document.querySelectorAll('.cell');
    if (!cells || cells.length === 0) {
        return;
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

// セルのクリックハンドラー
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
    bingoCard.innerHTML = '';
    window.marked = Array.from({ length: 5 }, () => Array(5).fill(false));
    data.forEach((row, i) => {
        for (let j = 0; j < 5; j++) {
            const cellDiv = document.createElement('div');
            cellDiv.className = 'cell';
            cellDiv.dataset.rowIndex = i;
            cellDiv.dataset.cellIndex = j;
            if (j < row.length) {
                cellDiv.textContent = row[j] !== 0 ? row[j] : '☆';
            } else {
                cellDiv.textContent = '';
            }
            if (j < row.length && row[j] !== 0 && isClickableCell(row[j])) {
                cellDiv.classList.add('clickable');
                cellDiv.addEventListener('click', cellClickHandler);
            }
            if (i === 2 && j === 2) {
                cellDiv.classList.add('clickable');
                cellDiv.addEventListener('click', cellClickHandler);
            }
            bingoCard.appendChild(cellDiv);

            if (!window.marked[i]) {
                window.marked[i] = [];
            }
            adjustFontSize(cellDiv);
        }
    });
    enableClickableCells();
}

// ビンゴをチェックする関数
function checkBingo() {
    fetch('/check-bingo', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ marked: window.marked }) // bingoCard ではなく marked を送信
    })
    .then(handleResponse)
    .then(data => {
        if (data.bingo) {
            alert('ビンゴです！');
        }
    })
    .catch(handleError);
}

// セルのフォントサイズを調整する関数
function adjustFontSize(cell) {
    const cellSize = Math.min(cell.offsetWidth, cell.offsetHeight);
    const fontSize = cellSize * 0.35;
    cell.style.fontSize = fontSize + "px";
}

// すべてのセルのフォントサイズを調整
function adjustAllCellFonts() {
    const cells = document.querySelectorAll(".cell");
    cells.forEach(function(cell) {
        adjustFontSize(cell);
    });
}

// 共通のレスポンスハンドラー
function handleResponse(response) {
    if (!response.ok) {
        return response.json().then(error => {
            throw new Error(error.message || 'Network response was not ok');
        });
    }
    return response.json();
}

