let ws;
let generateNumbersEnabled = false; // 初期状態はfalse
let roomPassword = ''; // グローバル変数としてパスワードを宣言

// セッションストレージに保存するキーを定義
const SESSION_STORAGE_KEY = 'bingoGameState';

// DOMが読み込まれた後にゲーム状態を復元する
document.addEventListener("DOMContentLoaded", function() {
    restoreGameStateFromSessionStorage(); // 先にゲーム状態を復元
    initializeWebSocket();
    adjustAllCellFonts();
    setupEventListeners();
    // setTimeout(restoreGameStateFromSessionStorage, 100); // 少し遅延させて復元
});

function saveGameStateToSessionStorage() {
    const gameState = {
        bingoCardState: serializeBingoCardState(), // ビンゴカードの状態を保存
        generatedNumbers: generatedNumbers, // 生成された数字の配列を保存
        roomPassword: roomPassword, // パスワードを保存
        styleState: serializeStyleState()  // スタイルの状態を保存
    };
    const serializedGameState = JSON.stringify(gameState);
    console.log("Saving to session storage:", serializedGameState); // 保存内容をコンソールに表示
    sessionStorage.setItem(SESSION_STORAGE_KEY, serializedGameState);
}

// セッションストレージから状態を復元する関数
function restoreGameStateFromSessionStorage() {
    const storedGameState = sessionStorage.getItem(SESSION_STORAGE_KEY);
    // console.log("Restoring from session storage:", storedGameState); 
    if (storedGameState) {
        const gameState = JSON.parse(storedGameState);
        console.log("Parsed game state:", gameState); // パースされた内容をコンソールに表示

        // ビンゴカードの状態を復元
        if (gameState.bingoCardState) {
            deserializeBingoCardState(gameState.bingoCardState);
        }

        // 生成された数字を復元
        if (gameState.generatedNumbers) {
            generatedNumbers = gameState.generatedNumbers;
            displayGeneratedNumbers(generatedNumbers); // 生成された数字を表示する
        }

        // パスワードを復元
        if (gameState.roomPassword) {
            roomPassword = gameState.roomPassword;
        }

        // スタイルの状態を復元
        if (gameState.styleState) {
            deserializeStyleState(gameState.styleState);
        }
    }
}

// ビンゴカードの状態をシリアライズする関数
function serializeBingoCardState() {
    const cells = document.querySelectorAll('.cell');
    const cellStates = [];
    cells.forEach(cell => {
        const state = {
            rowIndex: cell.dataset.rowIndex,
            cellIndex: cell.dataset.cellIndex,
            textContent: cell.textContent,
            className: cell.className,
            noneBlackState: cell.classList.contains('none-black')
        };
        cellStates.push(state);
    });
    return cellStates;
}

function deserializeBingoCardState(state) {
    console.log("Deserializing bingo card state:", state);
    state.forEach(cellState => {
        let cell = document.querySelector(`.cell[data-row-index="${cellState.rowIndex}"][data-cell-index="${cellState.cellIndex}"]`);
        if (!cell) {
            // セルが存在しない場合は新たに作成
            cell = document.createElement('div');
            cell.classList.add('cell');
            cell.dataset.rowIndex = cellState.rowIndex;
            cell.dataset.cellIndex = cellState.cellIndex;
            document.getElementById('bingo-card').appendChild(cell);
        }
        // console.log(`Updating cell at [${cellState.rowIndex}, ${cellState.cellIndex}] with:`, cellState);
        cell.textContent = cellState.textContent;
        cell.className = cellState.className;
        // none/blackの状態を反映
        if (cellState.noneBlackState) {
            cell.classList.add('none-black');
        } else {
            cell.classList.remove('none-black');
        }
    });
}
// 生成された数字を表示する関数
function displayGeneratedNumbers(numbers) {
    const generatedNumbersContainer = document.getElementById('log-container');
    generatedNumbersContainer.innerHTML = '';
    numbers.forEach(number => {
        const numberElement = document.createElement('div');
        numberElement.textContent = `ログ: ${number}`; // 追加のテキストを含める
        generatedNumbersContainer.appendChild(numberElement);
    });
}
// スタイルの状態を保存する関数
function serializeStyleState() {
    const row = document.querySelector('.row.mt-2');
    return {
        display: row ? row.style.display : ''
    };
}

// スタイルの状態を復元する関数
function deserializeStyleState(state) {
    const row = document.querySelector('.row.mt-2');
    if (row && state) {
        row.style.display = state.display;
    }
}

// ページをリロードする際にセッションストレージに状態を保存する
window.addEventListener('beforeunload', saveGameStateToSessionStorage);

// WebSocketの設定
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsHost = `${wsProtocol}//localhost:8080/ws`;

// WebSocketのイベントリスナーを定義
function initializeWebSocket() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsHost = `${wsProtocol}//localhost:8080/ws`;
    ws = new WebSocket(wsHost);

    ws.onopen = function(event) {
        console.log('WebSocket接続が確立された.');
        if (roomPassword) {
            ws.send(JSON.stringify({ type: 'join', password: roomPassword }));
        }
    };

    ws.onmessage = function(event) {
        console.log('WebSocketからデータを受信しました:', event.data);
        handleWebSocketMessage(event);
    };

    ws.onerror = function(error) {
        console.error('WebSocketエラー:', error);
    };

    ws.onclose = function(event) {
        console.log('WebSocket接続が閉じた:', event);
        setTimeout(initializeWebSocket, 1000); // 1秒後に再接続
    };
}

function handleWebSocketMessage(event) {
    try {
        const message = JSON.parse(event.data);
        if (message.type === 'number') {
            handleNewNumber(message.number);
        } else if (message.message) {
            console.log('Received message:', message.message);
        } else {
            console.error('Invalid message format received from WebSocket:', message);
        }
    } catch (error) {
        console.error('Error parsing WebSocket message:', error);
    }
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
    .then(handleResponse)
    .then(data => {
        if (data.message) {
            console.log(data.message); // 成功メッセージをコンソールに表示
            // パスワードが正しい場合の処理を追加
            fetchRoomNumbers(); // 成功した場合に、テキストファイルの情報を取得する処理を呼び出す
        }
    })
    .catch(handleError);
}

// ルーム作成リクエストをサーバーに送信
function createRoom() {
    // intervalInputから入力された値を整数に変換します
    const interval = parseInt(intervalInput.value);
    
    // 入力された値が有効な数値であり、0より大きい場合のみ処理を続行します
    if (!isNaN(interval) && interval > 0) {
        // fetch関数を使用して、サーバーの/create-roomエンドポイントにPOSTリクエストを送信します
        fetch('/create-room', {
            method: 'POST', // POSTメソッドを使用します
            headers: {
                'Content-Type': 'application/json' // リクエストのヘッダーでJSON形式のデータを指定します
            },
            body: JSON.stringify({ interval: interval }) // リクエストのボディにインターバル値を含めます
        })
        .then(response => response.json()) // レスポンスをJSON形式で解析します
        .then(data => {
            // サーバーから返ってきたデータ（ここではパスワード）を処理します
            if (data.password) {
                console.log(`生成されたパスワード: ${data.password}`); // コンソールにパスワードを表示します
                alert(`生成されたパスワード: ${data.password}`); // ユーザーにパスワードを示すアラートを表示します
            } else {
                alert('部屋のパスワードを生成できませんでした'); // パスワードが返されなかった場合のエラーアラートを表示します
            }
        })
        .catch(handleError); // エラーが発生した場合に処理するためのエラーハンドラーです
    } else {
        console.error('無効な間隔値:', intervalInput.value); // 入力されたインターバル値が無効な場合にコンソールにエラーメッセージを出力します
        alert('有効な間隔値を入力してください'); // ユーザーに有効なインターバル値を入力するように促すアラートを表示します
    }
}


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

    generateNumbersEnabled = true;
    // resetGame();
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

function fetchRoomNumbers() {
    console.log('Fetching room numbers...');

    const url = `/get-room-numbers?password=${encodeURIComponent(roomPassword)}`;

    fetch(url)
        .then(response => {
            if (!response.ok) {
                throw new Error(`部屋番号の取得に失敗しました。サーバーが返されました ${response.status} ${response.statusText}`);
            }
            // レスポンスをストリームとして受け取る
            const reader = response.body.getReader();

            // データを一つずつ処理する
            let decoder = new TextDecoder();
            let received = '';

            reader.read().then(function processText({ done, value }) {
                if (done) {
                    console.log('Received data:', received);
                    return;
                }
                
                received += decoder.decode(value, { stream: true });
                
                // データを改行で区切って分割する
                let parts = received.split('\n');
                parts.forEach(part => {
                    if (part.trim() !== '') {
                        try {
                            const jsonData = JSON.parse(part);
                            processReceivedData(jsonData);
                        } catch (error) {
                            console.error('Error parsing JSON:', error);
                            console.error('JSON parse error occurred in part:', part);
                        }
                    }
                });
                
                return reader.read().then(processText);
            });            
        })
        .catch(error => {
            console.error('部屋番号の取得中にエラーが発生しました:', error.message);
            alert('部屋番号を取得できませんでした。しばらくしてからもう一度お試しください。');
        });
}

function processReceivedData(data) {
    try {
        if (typeof data === 'number') {
            console.log('Received number:', data);
            handleNewNumber(data);
        } else if (Array.isArray(data)) {
            data.forEach(item => {
                if (typeof item === 'number') {
                    console.log('Received number:', item);
                    handleNewNumber(item);
                } else {
                    console.error('Invalid JSON data received:', item);
                }
            });
        } else {
            console.error('Invalid JSON data received:', data);
        }
    } catch (error) {
        console.error('Error handling new number data:', error.message);
        alert('サーバーからのデータ処理中にエラーが発生しました。しばらくしてからもう一度お試しください。');
    }
}

function handleNewNumber(number) {
    try {
        // すでにログに表示されている数字でない場合のみ表示する
        if (!generatedNumbers.includes(number)) {
            generatedNumbers.push(number);

            enableClickableCells();
            numberDiv.textContent = `Newナンバー: ${number}`;

            const logItem = document.createElement('div');
            logItem.textContent = `ログ: ${number}`;
            logDiv.appendChild(logItem);

            logDiv.scrollTop = logDiv.scrollHeight; // ログを常に最下部にスクロール

            playAudio(audioPath);

            // mute-toggleが存在する場合にのみ処理を行う
            const muteToggle = document.getElementById('mute-toggle');
            if (muteToggle && muteToggle.checked) {
                const audio = new Audio(audioPath);
                audio.play().catch(error => console.error('Error playing audio:', error));
            }
        }
    } catch (error) {
        console.error('Error handling new number data:', error.message);
        alert('サーバーからのデータ処理中にエラーが発生しました。しばらくしてからもう一度お試しください。');
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
