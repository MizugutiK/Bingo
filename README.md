# Bingo
Goを使ったバックエンドの練習

```mermaid
sequenceDiagram;
participant ホストユーザーフロント
participant ゲストユーザーフロント
<!-- participant フロント -->
participant サーバー

ホストユーザーフロント  ->>     サーバー:ルーム制作依頼
サーバー               -->>     ホストユーザーフロント:ルームパスワード送信
ゲストユーザーフロント  ->>     サーバー:パスワード入力
サーバー               -->>     ホストユーザーフロント:パスワード結果
alt パスワード一致　
フロント　-->> ゲストユーザーフロント:ルーム入出通知
else パスワード不一致
フロント -->> ゲストユーザーフロント:エラーを通知
end
ゲストユーザーフロント  ->>     サーバー:名前入力
サーバー               -->>      ゲストユーザーフロント:名前表示
ホストユーザーフロント  ->>     サーバー:数字入手リクエスト
サーバー               -->>     ホストユーザーフロント:数字表示
サーバー               -->>     ゲストユーザーフロント:数字表示
```

ゲームエンジンのプログラムを読んでおく