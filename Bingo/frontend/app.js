let ws;
let generateNumbersEnabled = false; // 初期状態はfalse
let roomPassword = ''; // グローバル変数としてパスワードを宣言
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
const resetButton = document.getElementById('reset-game');
const joinRoomButton = document.getElementById('join-room');
const createRoomButton = document.getElementById('create-room');
const roomTypeSelect = document.getElementById('room-type');
const setIntervalBtn = document.getElementById('set-interval-btn');
const intervalInput = document.getElementById('interval');
// UI周りの表示非表示用の宣言
const elementsToHide = document.querySelectorAll('#interval, #set-interval-btn, #CreateRoom, #join-room-container,#reset-game,#interval-label');
const password = document.getElementById('room-password').value

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

function joinRoom() {
    const password = document.getElementById('room-password').value; // パスワードを取得
    roomPassword = password; // グローバル変数にパスワードを保存
    // サーバーにパスワードを送信するためのリクエストを作成
    fetch('/join-room', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ password: password })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to join room. Server returned ' + response.status + ' ' + response.statusText);
        }
        return response.json();
    })
    .then(data => {
        if (data.message) {
            console.log(data.message); // 成功メッセージをコンソールに表示
            // パスワードが正しい場合の処理を追加
            fetchRoomNumbers(); // 成功した場合に、テキストファイルの情報を取得する処理を呼び出す
            // その他の処理を追加
        }
    })
    .catch(error => {
        console.error('Error:', error.message);
        alert(`Error: ${error.message}`);
    });
}

// ルーム作成リクエストをサーバーに送信
function createRoom() {

        fetch('/create-room', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
           
        })
        .then(response => response.json())
        .then(data => {
            if (data.password) {
                console.log(`Generated Password room: ${data.password}`);
                alert(`Generated Password room: ${data.password}`);
            } else {
                alert('Failed to generate room password');
            }
        })
        .catch(handleError);
}

// インターバル設定ボタンのクリックイベントリスナー
function handleSetIntervalBtnClick() {
    elementsToHide.forEach(element => {
        element.style.display = 'none';
    });
    document.querySelector('.row.mt-2').style.display = 'block';

    fetch('/new-game')
        .then(response => response.json())
        .then(data => {
            renderBingoCard(data);
            const interval = data.interval !== undefined ? data.interval : 1;
            startCountdown(interval);
        })
        .catch(handleError);

    const newInterval = parseInt(intervalInput.value);
    if (!isNaN(newInterval) && newInterval > 0) {
        fetch('/set-interval', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ interval: newInterval })
        })
        .then(response => response.text())
        .then(text => {
            try {
                const data = JSON.parse(text);
                console.log('Interval updated successfully', data);
                startCountdown(newInterval);
                generateNumbersEnabled = true;
                resetGame();
            } catch (error) {
                if (text === 'Interval has been set') {
                    console.log('数字生成間隔更新');
                    startCountdown(newInterval);
                    generateNumbersEnabled = true;
                    resetGame();
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

function fetchRoomNumbers() {
    console.log('Fetching room numbers...');

    // パスワードを使用してサーバーにリクエストを送信
    fetch(`/get-room-numbers?password=${encodeURIComponent(roomPassword)}`)
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to fetch room numbers. Server returned ' + response.status + ' ' + response.statusText);
            }
            return response.json();
        })
        .then(data => {
            if (data && data.numbers) {
                handleNewNumber(data.numbers);
            } else {
                throw new Error('Invalid response format: missing numbers');
            }
        })
        .catch(error => {
            console.error('Error fetching room numbers:', error.message);
            alert('Failed to fetch room numbers. Please try again later.');
        });
}

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
    playAudio(audioPath);
    if (document.getElementById('mute-toggle').checked) {
        const audio = new Audio(audioPath);
        audio.play().catch(error => console.error('Error playing audio:', error));
    }
}


// カウントダウンを開始する関数
function startCountdown(interval) {
    if (!interval || typeof interval !== 'number') {
        console.error('Invalid interval value:', interval);
        return;
    }

    if (countdownInterval) {
        clearInterval(countdownInterval);
    }

    countdownDiv.textContent = interval;

    countdownInterval = setInterval(() => {
        let currentCount = parseInt(countdownDiv.textContent);
        if (isNaN(currentCount)) {
            currentCount = interval;
        }

        currentCount--;

        if (currentCount <= 0) {
            currentCount = interval;
          
        }

        countdownDiv.textContent = currentCount;
    }, 1000);
}


// セルのクリックハンドラー
function cellClickHandler() {
    const rowIndex = parseInt(this.dataset.rowIndex);
    const cellIndex = parseInt(this.dataset.cellIndex);
    window.marked[rowIndex][cellIndex] = !window.marked[rowIndex][cellIndex];
    this.classList.toggle('marked');
    checkBingo();
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
            cell.removeEventListener('click', cellClickHandler); // イベントリスナーを削除
        }
    });

    const centerCell = document.querySelector('[data-row-index="2"][data-cell-index="2"]');
    if (centerCell) {
        centerCell.classList.add('clickable');
        centerCell.addEventListener('click', cellClickHandler);
    }
}

// SEを再生する関数
function playAudio(audioPath) {
    const audio = new Audio(audioPath);
    audio.play();
}

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

// セルをマークする関数
function markCell(cellElement) {
    if (!cellElement || !cellElement.textContent) {
        return;
    }

    const value = parseInt(cellElement.textContent);
    if (isNaN(value) || generatedNumbers.indexOf(value) === -1) {
        return;
    }

    cellElement.classList.add('marked');
    const rowIndex = Array.from(cellElement.parentNode.parentNode.children).indexOf(cellElement.parentNode);
    const cellIndex = Array.from(cellElement.parentNode.children).indexOf(cellElement);
    window.marked[rowIndex][cellIndex] = true;

    if (checkBingo()) {
        alert('Bingo!');
    }
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

// フェッチレスポンスを処理する関数
function handleResponse(response) {
    if (!response.ok) {
        throw new Error('Network response was not ok.');
    }
    return response.json();
}

// ルーム参加のレスポンスを処理する関数
function handleJoinRoomResponse(data) {
    if (!data || !data.card || !Array.isArray(data.card)) {
        throw new Error('Invalid response data');
    }
    renderBingoCard(data);
    startCountdown(data.interval);
}

// ルーム参加エラーを処理する関数
function handleJoinRoomError(error) {
    console.error('ルームに参加できませんでした:', error.message);
    alert('ルームに参加できませんでした。再試行してください。');
}
