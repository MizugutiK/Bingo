# Bingo
Goを使ったバックエンドの練習

```mermaid
sequenceDiagram;
participant ホストユーザー
participant ゲストユーザー
participant フロント
participant サーバー

ホストユーザー　>>　フロント:ルーム制作を行う
フロント　>>　サーバー:ルーム制作依頼
サーバー　>>　フロント:ルームパスワード送信
フロント　>>　ホストユーザー: パスワード表示
ゲストユーザー >> フロント:パスワード入力
フロント >> サーバー:パスワード照合
サーバー >> フロント:結果
alt パスワード一致　
フロント　>> ゲストユーザー:ルーム入出通知
else パスワード不一致
フロント >> ゲストユーザー:エラーを通知
end


```