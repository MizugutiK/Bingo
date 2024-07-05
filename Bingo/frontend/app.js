let ws; // WebSocketインスタンスを保持する変数
let generateNumbersEnabled = false; // 数字生成が有効かどうかのフラグ。初期状態はfalse
let roomPassword = ''; // ルームのパスワードをグローバル変数として宣言

// セッションストレージに保存するキーを定義
const SESSION_STORAGE_KEY = 'bingoGameState';

// DOMのロード完了後に実行される初期化関数
document.addEventListener("DOMContentLoaded", function() {
    restoreGameStateFromSessionStorage(); // ゲーム状態をセッションストレージから復元
    initializeWebSocket(); // WebSocketを初期化
    adjustAllCellFonts(); // 全てのセルのフォントを調整
    setupEventListeners(); // イベントリスナーを設定
});

// ゲーム状態をセッションストレージに保存する関数
function saveGameStateToSessionStorage() {
    const gameState = {
        bingoCardState: serializeBingoCardState(), // ビンゴカードの状態をシリアライズして保存
        generatedNumbers: generatedNumbers, // 生成された数字の配列を保存
        roomPassword: roomPassword, // ルームのパスワードを保存
        styleState: serializeStyleState()  // スタイルの状態をシリアライズして保存
    };
    const serializedGameState = JSON.stringify(gameState); // ゲーム状態をJSON文字列に変換
    console.log("Saving to session storage:", serializedGameState); // 保存内容をコンソールに表示
    sessionStorage.setItem(SESSION_STORAGE_KEY, serializedGameState); // セッションストレージに保存
}

// セッションストレージから状態を復元する関数
function restoreGameStateFromSessionStorage() {
    const storedGameState = sessionStorage.getItem(SESSION_STORAGE_KEY); // セッションストレージからゲーム状態を取得
    // console.log("Restoring from session storage:", storedGameState);
    if (storedGameState) {
        const gameState = JSON.parse(storedGameState); // JSON文字列をパースしてゲーム状態オブジェクトに変換
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

        // ルームのパスワードを復元
        if (gameState.roomPassword) {
            roomPassword = gameState.roomPassword;
        }

        // スタイルの状態を復元
        if (gameState.styleState) {
            deserializeStyleState(gameState.styleState);
        }
    }
}

// ビンゴカードを保存可能な形式にする関数
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

// ビンゴカードの状態を復元する関数
function deserializeBingoCardState(state) {
    // console.log("Deserializing bingo card state:", state);
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
        // セルの内容とスタイルを更新
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
        numberElement.textContent = `ログ: ${number}`; // ログの形式で表示
        generatedNumbersContainer.appendChild(numberElement);
    });
}

// スタイルの状態を保存する関数
function serializeStyleState() {
    // const row = document.querySelector('.row.mt-2');
    return {
        display: row ? row.style.display : ''
    };
}

// スタイルの状態を復元する関数
function deserializeStyleState(state) {
    // const row = document.querySelector('.row.mt-2');
    if (row && state) {
        row.style.display = state.display;
    }
}

// ページをリロードする際にセッションストレージに状態を保存する
window.addEventListener('beforeunload', saveGameStateToSessionStorage);

// WebSocketの設定
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsHost = `${wsProtocol}//localhost:8080/ws`;

// WebSocketの初期化とイベントリスナーの設定
function initializeWebSocket() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsHost = `${wsProtocol}//localhost:8080/ws`;
    ws = new WebSocket(wsHost);

    ws.onopen = function(event) {
        console.log('WebSocket接続が確立された.');
        // パスワードが設定されていればルームに参加
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
        setTimeout(initializeWebSocket, 1000); // 1秒後に再接続を試行
    };
}

// WebSocketメッセージの処理
function handleWebSocketMessage(event) {
    try {
        const message = JSON.parse(event.data);
        if (message.type === 'number') {
            handleNewNumber(message.number); // 新しい数字を処理
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
const bingoCard = document.getElementById('bingo-card'); // ビンゴカード要素
const numberDiv = document.getElementById('number'); // 数字表示用要素
const countdownDiv = document.getElementById('countdown'); // カウントダウン表示用要素
const logDiv = document.getElementById('log'); // ログ表示用要素
const resetButton = document.getElementById('reset-game'); // リセットボタン要素
const joinRoomButton = document.getElementById('join-room'); // ルーム参加ボタン要素
const createRoomButton = document.getElementById('create-room'); // ルーム作成ボタン要素
const roomTypeSelect = document.getElementById('room-type'); // ルームタイプ選択要素
const setIntervalBtn = document.getElementById('set-interval-btn'); // インターバル設定ボタン要素
const intervalInput = document.getElementById('interval'); // インターバル入力要素
const row = document.querySelector('.row.mt-2');
// UI周りの表示非表示用の宣言
const elementsToHide = document.querySelectorAll('#interval, #set-interval-btn, #CreateRoom, #join-room-container,#reset-game,#interval-label');
const password = document.getElementById('room-password').value

// ビンゴカードを非表示にする
row.style.display = 'none';

// イベントリスナーの設定
function setupEventListeners() {
    resetButton.addEventListener('click', resetGame); // リセットボタンのクリックイベント
    joinRoomButton.addEventListener('click', joinRoom); // ルーム参加ボタンのクリックイベント
    createRoomButton.addEventListener('click', createRoom); // ルーム作成ボタンのクリックイベント
    setIntervalBtn.addEventListener('click', handleSetIntervalBtnClick); // インターバル設定ボタンのクリックイベント
    window.addEventListener("resize", adjustAllCellFonts); // ウィンドウのリサイズイベント
}

// リセットボタンのクリックイベントリスナー
function resetGame() {
    clearInterval(countdownInterval); // カウントダウンのインターバルをクリア
    countdownDiv.textContent = ''; // カウントダウン表示をクリア
    console.log('番号リセット');
    fetch('/reset-generated-numbers') // 生成された数字をリセットするためのリクエストを送信
        .then(handleResponse)
        .then(() => {
            generatedNumbers = [];
            numberDiv.textContent = ''; // 数字表示をクリア
            logDiv.innerHTML = ''; // ログ表示をクリア
            countdownDiv.textContent = ''; // カウントダウン表示をクリア
            resetButton.style.display = 'none'; // リセットボタンを非表示
            console.log('リセットできた');
        })
        .catch(handleError);
}

// ルーム参加ボタンのクリックイベントリスナー
function joinRoom() {
    const password = document.getElementById('room-password').value; // 入力されたパスワードを取得
    roomPassword = password; // グローバル変数にパスワードを保存
    // サーバーにパスワードを送信するリクエストを作成
    fetch('/join-room', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ password: password }) // パスワードをJSON形式で送信
    })
    .then(handleResponse)
    .then(data => {
        if (data.message) {
            console.log(data.message); // 成功メッセージをコンソールに表示
            // パスワードが正しい場合の処理を追加
            fetchRoomNumbers(); // 成功した場合に、テキストファイルの情報を取得する処理を呼び出す
        }
    })
    .catch(handleError); // エラーハンドリング
}

// ルーム作成ボタンのクリックイベントリスナー
function createRoom() {
    const interval = parseInt(intervalInput.value); // 入力されたインターバル値を整数に変換
    
    // 有効なインターバル値かどうかをチェック
    if (!isNaN(interval) && interval > 0) {
        // サーバーに/create-roomエンドポイントにPOSTリクエストを送信
        fetch('/create-room', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json' // JSON形式のデータを送信する
            },
            body: JSON.stringify({ interval: interval }) // リクエストボディにインターバル値を含める
        })
        .then(response => response.json()) // レスポンスをJSON形式で解析
        .then(data => {
            if (data.password) {
                console.log(`生成されたパスワード: ${data.password}`); // 生成されたパスワードを表示
                alert(`生成されたパスワード: ${data.password}`); // ユーザーに生成されたパスワードを示すアラートを表示
            } else {
                alert('部屋のパスワードを生成できませんでした'); // パスワードが生成されなかった場合のエラーアラート
            }
        })
        .catch(handleError); // エラーハンドリング
    } else {
        console.error('無効な間隔値:', intervalInput.value); // 無効なインターバル値が入力された場合のコンソールメッセージ
        alert('有効な間隔値を入力してください'); // ユーザーに有効なインターバル値を入力するように促すアラート
    }
}

// インターバル設定ボタンのクリックイベントを処理する関数
function handleSetIntervalBtnClick() {
    // 非表示にする要素を非表示にする
    elementsToHide.forEach(element => {
        element.style.display = 'none';
    });
    // ビンゴカードを表示する
  row.style.display = 'block';

    // 新しいゲームの開始をサーバーに要求し、ビンゴカードをレンダリングする
    fetch('/new-game')
        .then(response => response.json())
        .then(data => {
            renderBingoCard(data); // ビンゴカードをレンダリングする
            const interval = data.interval !== undefined ? data.interval : 1; // 取得したインターバルを設定し、デフォルト値は1
            startCountdown(interval); // カウントダウンを開始する
        })
        .catch(handleError); // エラーハンドリング

    generateNumbersEnabled = true; // 数字の生成を有効にする
    // resetGame(); // ゲームをリセットする（コメントアウトされているが、必要に応じて使用する）
}

// 共通のエラーハンドラー関数
function handleError(error) {
    console.error('Error:', error.message); // エラーメッセージをコンソールに出力する
    alert(`Error: ${error.message}`); // エラーメッセージをアラートで表示する
}

let countdownInterval;
let generatedNumbers = [];
const audioPath = 'chime.mp3';

// セルがクリック可能かどうかを判断する関数
function isClickableCell(cellValue) {
    return !generatedNumbers.includes(cellValue); // 生成された数字に含まれていなければクリック可能
}

// ルーム番号を取得する関数
function fetchRoomNumbers() {
    console.log('Fetching room numbers...'); // ルーム番号の取得を開始するログメッセージ

    const url = `/get-room-numbers?password=${encodeURIComponent(roomPassword)}`; // パスワードを含むURLを生成する

    fetch(url)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Failed to fetch room numbers. Server returned ${response.status} ${response.statusText}`); // エラーメッセージを投げる
            }
            // レスポンスをストリームとして処理する
            const reader = response.body.getReader();

            // データを一つずつ処理する
            let decoder = new TextDecoder();
            let received = '';

            reader.read().then(function processText({ done, value }) {
                if (done) {
                    console.log('Received data:', received); // 受信したデータをログに出力する
                    return;
                }
                
                received += decoder.decode(value, { stream: true }); // 受信したデータをデコードする
                
                // データを改行で区切って分割する
                let parts = received.split('\n');
                parts.forEach(part => {
                    if (part.trim() !== '') {
                        try {
                            const jsonData = JSON.parse(part); // JSONデータを解析する
                            processReceivedData(jsonData); // 受信したデータを処理する
                        } catch (error) {
                            console.error('Error parsing JSON:', error); // JSON解析エラーをコンソールに出力する
                            console.error('JSON parse error occurred in part:', part); // エラーが発生した部分の内容をコンソールに出力する
                        }
                    }
                });
                
                return reader.read().then(processText); // 次のデータを処理する
            });            
        })
        .catch(error => {
            console.error('Error fetching room numbers:', error.message); // ルーム番号の取得中にエラーが発生した場合のエラーメッセージをコンソールに出力する
            alert('Failed to fetch room numbers. Please try again later.'); // ユーザーにエラーメッセージをアラートで表示する
        });
}
// 受信したデータを処理する関数
function processReceivedData(data) {
    try {
        if (typeof data === 'number') {
            console.log('Received number:', data);
            handleNewNumber(data); // 受信した数字を処理する
        } else if (Array.isArray(data)) {
            // データが配列の場合、各アイテムを処理する
            data.forEach(item => {
                if (typeof item === 'number') {
                    console.log('Received number:', item);
                    handleNewNumber(item); // 受信した各数字を処理する
                } else {
                    console.error('Invalid JSON data received:', item); // 数字以外の無効なJSONデータをエラーログに出力する
                }
            });
        } else {
            console.error('Invalid JSON data received:', data); // 数字でも配列でもない無効なJSONデータをエラーログに出力する
        }
    } catch (error) {
        console.error('Error handling new number data:', error.message); // 新しい数字データの処理中にエラーが発生した場合のエラーメッセージをコンソールに出力する
        alert('サーバーからのデータ処理中にエラーが発生しました。しばらくしてからもう一度お試しください。'); // ユーザーにエラーメッセージをアラートで表示する
    }
}

// 新しい数字を処理する関数
function handleNewNumber(number) {
    try {
        // すでにログに表示されている数字でない場合のみ処理する
        if (!generatedNumbers.includes(number)) {
            generatedNumbers.push(number); // 生成された数字を配列に追加する

            enableClickableCells(); // クリック可能なセルを有効にする
            numberDiv.textContent = `Newナンバー: ${number}`; // 新しい数字を表示する

            const logItem = document.createElement('div');
            logItem.textContent = `ログ: ${number}`; // ログアイテムに数字を表示する
            logDiv.appendChild(logItem); // ログにログアイテムを追加する

            logDiv.scrollTop = logDiv.scrollHeight; // ログを常に最下部にスクロールする

            playAudio(audioPath); // オーディオを再生する

            // mute-toggleが存在する場合にのみ処理を行う
            const muteToggle = document.getElementById('mute-toggle');
            if (muteToggle && muteToggle.checked) {
                const audio = new Audio(audioPath);
                audio.play().catch(error => console.error('Error playing audio:', error)); // オーディオ再生中にエラーが発生した場合のエラーログを出力する
            }
        }
    } catch (error) {
        console.error('Error handling new number data:', error.message); // 新しい数字データの処理中にエラーが発生した場合のエラーメッセージをコンソールに出力する
        alert('サーバーからのデータ処理中にエラーが発生しました。しばらくしてからもう一度お試しください。'); // ユーザーにエラーメッセージをアラートで表示する
    }
}

// カウントダウンを開始する関数
function startCountdown(interval) {
    if (!interval || typeof interval !== 'number') {
        console.error('Invalid interval value:', interval); // 無効なインターバル値をエラーログに出力する
        return;
    }

    if (countdownInterval) {
        clearInterval(countdownInterval); // 既存のカウントダウンをクリアする
    }

    countdownDiv.textContent = interval; // カウントダウン用の要素に初期値を設定する

    countdownInterval = setInterval(() => {
        let currentCount = parseInt(countdownDiv.textContent);
        if (isNaN(currentCount)) {
            currentCount = interval;
        }

        currentCount--;

        if (currentCount <= 0) {
            currentCount = interval; // カウントダウンが0以下になった場合、再度インターバル値にリセットする
        }

        countdownDiv.textContent = currentCount; // 現在のカウントを画面に表示する
    }, 1000); // 1000ミリ秒（1秒）ごとに更新する
}

// セルのクリックハンドラー
function cellClickHandler() {
    const rowIndex = parseInt(this.dataset.rowIndex); // クリックされたセルの行インデックスを取得
    const cellIndex = parseInt(this.dataset.cellIndex); // クリックされたセルの列インデックスを取得
    window.marked[rowIndex][cellIndex] = !window.marked[rowIndex][cellIndex]; // クリックされたセルのマーク状態を反転させる
    this.classList.toggle('marked'); // セルに'marked'クラスをトグルする（マーク表示を切り替える）
    checkBingo(); // ビンゴ状態をチェックする
}

// セルがクリック可能になるかどうかを判断する関数
function enableClickableCells() {
    const cells = document.querySelectorAll('.cell'); // 全てのセルを取得する
    if (!cells || cells.length === 0) {
        return;
    }
    cells.forEach(cell => {
        const cellNumber = cell.textContent === 'FREE' ? 0 : parseInt(cell.textContent); // セルの内容を数値に変換する（'FREE'の場合は0）
        if (Array.isArray(generatedNumbers) && (generatedNumbers.includes(cellNumber) || cellNumber === 0)) {
            // 生成された数字の配列に含まれているか、セルの数字が0（FREEセル）の場合
            if (!cell.classList.contains('clickable')) {
                cell.classList.add('clickable'); // 'clickable'クラスを追加してセルをクリック可能にする
                cell.addEventListener('click', cellClickHandler); // クリックイベントリスナーを追加
            }
        } else {
            cell.classList.remove('clickable'); // 'clickable'クラスを削除してセルをクリック不可にする
            cell.removeEventListener('click', cellClickHandler); // クリックイベントリスナーを削除
        }
    });

    const centerCell = document.querySelector('[data-row-index="2"][data-cell-index="2"]');
    if (centerCell) {
        centerCell.classList.add('clickable'); // 中央セルをクリック可能にする
        centerCell.addEventListener('click', cellClickHandler); // 中央セルにクリックイベントリスナーを追加
    }
}

// SEを再生する関数
function playAudio(audioPath) {
    const audio = new Audio(audioPath); // 指定されたパスのオーディオを作成する
    audio.play(); // オーディオを再生する
}

// ビンゴカードをレンダリングする関数
function renderBingoCard(data) {
    bingoCard.innerHTML = ''; // ビンゴカードをクリアする
    window.marked = Array.from({ length: 5 }, () => Array(5).fill(false)); // マークされた状態を管理する配列を初期化する
    data.forEach((row, i) => {
        for (let j = 0; j < 5; j++) {
            const cellDiv = document.createElement('div'); // 新しいセル要素を作成する
            cellDiv.className = 'cell'; // セルに'class'属性を追加する
            cellDiv.dataset.rowIndex = i; // 行インデックスをデータ属性としてセットする
            cellDiv.dataset.cellIndex = j; // 列インデックスをデータ属性としてセットする
            if (j < row.length) {
                cellDiv.textContent = row[j] !== 0 ? row[j] : '☆'; // セルのテキストコンテンツを設定する（'FREE'セルは'☆'にする）
            } else {
                cellDiv.textContent = ''; // 空のセルの場合、テキストコンテンツをクリアする
            }
            if (j < row.length && row[j] !== 0 && isClickableCell(row[j])) {
                cellDiv.classList.add('clickable'); // クリック可能なセルに'class'属性を追加する
                cellDiv.addEventListener('click', cellClickHandler); // クリックイベントリスナーを追加する
            }
            if (i === 2 && j === 2) {
                cellDiv.classList.add('clickable'); // 中央セルに'class'属性を追加する
                cellDiv.addEventListener('click', cellClickHandler); // クリックイベントリスナーを追加する
            }
            bingoCard.appendChild(cellDiv); // セルをビンゴカードに追加する

            if (!window.marked[i]) {
                window.marked[i] = [];
            }
            adjustFontSize(cellDiv); // セルのフォントサイズを調整する
        }
    });
    enableClickableCells(); // クリック可能なセルを有効にする
}

// セルをマークする関数
function markCell(cellElement) {
    if (!cellElement || !cellElement.textContent) {
        return; // セルが存在しないか、テキスト内容がない場合は処理しない
    }

    const value = parseInt(cellElement.textContent);
    if (isNaN(value) || generatedNumbers.indexOf(value) === -1) {
        return; // 数値に変換できない場合や、生成された数字のリストに含まれていない場合は処理しない
    }

    cellElement.classList.add('marked'); // セルに'marked'クラスを追加する（マーク表示を有効にする）
    const rowIndex = Array.from(cellElement.parentNode.parentNode.children).indexOf(cellElement.parentNode);
    const cellIndex = Array.from(cellElement.parentNode.children).indexOf(cellElement);
    window.marked[rowIndex][cellIndex] = true; // マークされたセルを記録する

    if (checkBingo()) { // ビンゴをチェックする
        alert('Bingo!'); // ビンゴが成立した場合にアラートを表示する
    }
}

// ビンゴをチェックする関数
function checkBingo() {
    fetch('/check-bingo', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ marked: window.marked }) // マークされたセルの状態をサーバーに送信する
    })
    .then(handleResponse) // レスポンスを処理する
    .then(data => {
        if (data.bingo) {
            alert('ビンゴです！'); // サーバーからのレスポンスでビンゴが成立している場合にアラートを表示する
        }
    })
    .catch(handleError); // エラーが発生した場合にエラーハンドラーを実行する
}

// セルのフォントサイズを調整する関数
function adjustFontSize(cell) {
    const cellSize = Math.min(cell.offsetWidth, cell.offsetHeight); // セルの幅と高さの小さい方を取得する
    const fontSize = cellSize * 0.35; // フォントサイズをセルサイズに基づいて計算する
    cell.style.fontSize = fontSize + "px"; // セルのフォントサイズを設定する
}

// すべてのセルのフォントサイズを調整する関数
function adjustAllCellFonts() {
    const cells = document.querySelectorAll(".cell"); // 全てのセルを取得する
    cells.forEach(function(cell) {
        adjustFontSize(cell); // 各セルのフォントサイズを調整する
    });
}

// フェッチレスポンスを処理する関数
function handleResponse(response) {
    if (!response.ok) {
        throw new Error('Network response was not ok.'); // レスポンスが成功していない場合はエラーをスローする
    }
    return response.json(); // JSON形式のレスポンスを返す
}

// ルーム参加のレスポンスを処理する関数
function handleJoinRoomResponse(data) {
    if (!data || !data.card || !Array.isArray(data.card)) {
        throw new Error('Invalid response data'); // レスポンスデータが無効な場合はエラーをスローする
    }
    renderBingoCard(data); // ビンゴカードをレンダリングする
    startCountdown(data.interval); // カウントダウンを開始する
}

// ルーム参加エラーを処理する関数
function handleJoinRoomError(error) {
    console.error('ルームに参加できませんでした:', error.message); // エラーメッセージをコンソールに出力する
    alert('ルームに参加できませんでした。再試行してください。'); // アラートを表示してユーザーに通知する
}
