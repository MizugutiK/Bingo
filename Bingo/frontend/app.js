let ws;

document.addEventListener("DOMContentLoaded", function() {
    const cells             = document.querySelectorAll(".cell");
    // すべてのセルのフォントサイズを調整
    function    adjustAllCellFonts() {
        cells.forEach(function(cell) {
            adjustFontSize(cell);
        });
    }
    // ウィンドウサイズが変更されたときにフォントサイズを再調整
    window.addEventListener("resize", adjustAllCellFonts);
    // セルのフォントサイズを調整する関数
    function    adjustFontSize(cell) {
        const cellSize      = Math.min(cell.offsetWidth, cell.offsetHeight);
        const fontSize      = cellSize * 0.35;
        cell.style.fontSize = fontSize + "px";
    }
    adjustAllCellFonts(); // 初期化時にフォントサイズを調整

    // WebSocketの設定
    const wsProtocol        = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsHost            = `${wsProtocol}//localhost:8080/ws`;
    // WebSocketのイベントリスナーを定義
    function    initializeWebSocket() {
        ws = new WebSocket(wsHost);

        ws.onopen = function(event) {
            console.log     ('WebSocket接続が確立された.');
        };

        ws.onmessage = function(event) {
            const generatedNumbersFromServer    = JSON.parse(event.data);
            generatedNumbers                    = generatedNumbersFromServer;
            enableClickableCells();
            handleNewNumber(event.data);
        };

        ws.onerror = function(error) {
            console.error(' WebSocketエラー:', error);
        };

        ws.onclose = function(event) {
            console.log     ('WebSocket接続が閉じた:', event);
        };
    }
    initializeWebSocket();

    // 必要な要素を取得

    const bingoCard         = document.getElementById('bingo-card');
    const numberDiv         = document.getElementById('number');
    const countdownDiv      = document.getElementById('countdown');
    const logDiv            = document.getElementById('log');
    // ルーム名を表示する要素を取得
    const roomNameDisplay   = document.getElementById('roomNameDisplay');

    const newgameButton     = document.getElementById('new-game');
    const resetButton       = document.getElementById('reset-game');
    const joinRoomButton    = document.getElementById('join-room');
    const roomNameInput     = document.getElementById('roomname');
    const createRoomButton  = document.getElementById('create-room');
    // ルームタイプの選択要素を取得
    const roomTypeSelect    = document.getElementById('room-type');

    // 新しいゲームを開始するボタンのクリックイベントリスナー
    newgameButton.addEventListener      ('click', startNewGame);
    // リセットボタンのクリックイベントリスナー
    resetButton.addEventListener        ('click', resetGame);
    // ルーム参加ボタンにクリックイベントリスナー
    joinRoomButton.addEventListener     ('click', joinRoom);
    // ルームを立てるクリックイベントリスナー
    createRoomButton.addEventListener   ('click', createRoom);


    // 新しいゲームを開始する関数
    function    startNewGame() {
        fetch('/new-game')
            .then(response => response.json())
            .then(renderNewGame)
            .catch(error => console.error('Error:', error));
    }

    // リセットボタンのクリックイベントリスナーを追加
    function    resetGame() {
        fetch('/reset-generated-numbers')
        .then(response => {
            if (response.ok) {
                generatedNumbers = [];
                numberDiv.textContent = '';
                logDiv.innerHTML = '';
                clearInterval(countdownInterval);
                countdownDiv.textContent = '';
                resetButton.style.display = 'none';
                newgameButton.style.display = "block";
            } else {
                console.error('Failed to reset generated numbers');
            }
        })
        .catch(error => console.error('Error:', error));
    }

    // ルームに参加する関数
    function    joinRoom() {
        const roomName = roomNameInput.value.trim(); 
        const password      = document.getElementById('room-password').value;
        console.log     ('Password:', password); // パスワードをログに出力

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
    function    createRoom() {
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
        .catch(error => {
            console.error('Error:', error.message);
        });
} else {
    // ルーム名が空の場合はアラートを表示するなど、適切な処理を行う
    alert('ルーム名を入力してください。');
}
}

    let countdownInterval;
    let countdownTime = 60;
    let generatedNumbers = [];
    const audioPath         = 'chime.mp3';


    // セルがクリック可能かどうかを判断する関数
    function    isClickableCell(cellValue) {
        return !generatedNumbers.includes(cellValue);
    }

    // 新しい数字を取得したときの処理
    function    handleNewNumber(data) {
    if (!data) {
        console.error('Empty message received from WebSocket.');
        return;
    }

    const numbers       = JSON.parse(data);
    if (!Array.isArray(numbers)) {
        console.error('Invalid message format received from WebSocket.');
        return;
    }

    const latestNumber  = numbers[numbers.length - 1];

    // generatedNumbersが配列でない場合は、新しい配列として初期化する
    if (!Array.isArray(generatedNumbers)) {
        generatedNumbers = [];
    }

    generatedNumbers.push(latestNumber);

    enableClickableCells(latestNumber);
    numberDiv.textContent = `Newナンバー: ${latestNumber}`;

    logDiv.innerHTML = '';
    numbers.forEach(number => {
        const logItem   = document.createElement('div');
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
    function    handleJoinRoomResponse(response) {
        console.log     ('Response status:', response.status);
        console.log     ('Response headers:', response.headers);
        return response.text().then(text => {
            console.log     ('Raw response text:', text); // レスポンスの生データをログに出力
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
    function    handleJoinRoomError(error) {
        console.error('Error:', error.message);
        alert(`Error: ${error.message}`);
    }

    // ルーム参加時のデータを処理する関数
    function    handleJoinRoomData(data) {
        console.log     ('Response data:', data);
        if (data.room_id) {
            alert(`Joined room ${data.room_id}`);
            ws = new WebSocket(`${wsHost}?room_id=${data.room_id}`);
    
            ws.onopen = () => {
                console.log     ('WebSocket connection established for room:', data.room_id);
            };
    
            ws.onmessage = event => {
                console.log     ('Received WebSocket message:', event.data);
                handleNewNumber(event.data);
            };
    
            ws.onerror = error => {
                console.error('WebSocket error:', error);
            };
    
            ws.onclose = event => {
                console.log     ('WebSocket connection closed:', event);
            };
        } else {
            alert('Failed to join room');
        }
    }

    // 新しいゲームのデータをレンダリングする関数
    function    renderNewGame(data) {
        renderBingoCard(data);
        startCountdown();
        generatedNumbers = data.generatedNumbers;
        enableClickableCells();
        resetButton.style.display = 'none';
        newgameButton.style.display = "none";
        if (createRoomButton) {
            createRoomButton.style.display = "none"; // createRoomButtonを参照
        }
    }

    // カウントダウンを開始する関数
    function    startCountdown() {
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
    function    updateCountdown() {
        const seconds           = countdownTime % 60;
        countdownDiv.textContent = `${seconds < 10 ? '0' : ''}${seconds}`;
    }

    // セルがクリック可能になるかどうかを判断する関数
    function    enableClickableCells() {
        const cells             = document.querySelectorAll('.cell');
        if (!cells || cells.length === 0) {
            return;
        }
        cells.forEach(cell => {
            const cellNumber    = cell.textContent === 'FREE' ? 0 : parseInt(cell.textContent);
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

        const centerCell        = document.querySelector('[data-row-index="2"][data-cell-index="2"]');
        centerCell.classList.add('clickable');
        centerCell.addEventListener('click', cellClickHandler);
    }

    // セルのクリックハンドラー
    function    cellClickHandler() {
        const rowIndex          = parseInt(this.dataset.rowIndex);
        const cellIndex         = parseInt(this.dataset.cellIndex);
        window.marked[rowIndex][cellIndex] = !window.marked[rowIndex][cellIndex];
        this.classList.toggle('marked');
        checkBingo();
    }

    // SEを再生する関数
    function    playAudio(audioPath) {
        const audio             = new Audio(audioPath);
        audio.play();
    }

    // ビンゴカードをレンダリングする関数
    function    renderBingoCard(data) {
        bingoCard.innerHTML = '';
        window.marked = Array.from({ length: 5 }, () => Array(5).fill(false));
        data.forEach((row, i) => {
            for (let j = 0; j < 5; j++) {
                const cellDiv   = document.createElement('div');
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
    function    checkBingo() {
        fetch('/check-bingo', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ marked: window.marked }) // bingoCard ではなく marked を送信
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.json();
        })
        .then(data => {
            if (data.bingo) {
                alert('ビンゴです！');
                newgameButton.style.display = "block";
            }
        })
        .catch(error => console.error('Error:', error.message));
    }

}); 
