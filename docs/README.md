# Himasoku ドキュメント

「暇」をグループ内で共有し、プッシュ通知で誘い合うアプリのバックエンド（Go 版）の設計ドキュメント。

既存の Rails 実装（[`sawakishuto/HimasokuBackend`](https://github.com/sawakishuto/HimasokuBackend)）の
仕様を引き継ぎつつ、スキーマ設計を見直している。各ユースケースには
**既存実装との差分** を併記しているので、移行時の影響範囲はそこを見れば分かる。

## ユースケース一覧

| ID | ユースケース | ドキュメント |
|----|-------------|-------------|
| UC-01 | サインインとユーザーの自動プロビジョニング | [オンボーディング](usecases/onboarding.md#uc-01-サインインとユーザーの自動プロビジョニング) |
| UC-02 | デバイストークンの登録 | [オンボーディング](usecases/onboarding.md#uc-02-デバイストークンの登録) |
| UC-03 | プロフィールの取得 | [オンボーディング](usecases/onboarding.md#uc-03-プロフィールの取得) |
| UC-04 | グループの作成 | [グループ](usecases/groups.md#uc-04-グループの作成) |
| UC-05 | 招待コードでグループに参加 | [グループ](usecases/groups.md#uc-05-招待コードでグループに参加) |
| UC-06 | 所属グループの一覧 | [グループ](usecases/groups.md#uc-06-所属グループの一覧) |
| UC-07 | グループメンバーの一覧 | [グループ](usecases/groups.md#uc-07-グループメンバーの一覧) |
| UC-08 | グループからの退出 | [グループ](usecases/groups.md#uc-08-グループからの退出) |
| UC-09 | 暇を共有する | [暇の共有](usecases/availability-sharing.md#uc-09-暇を共有する) |
| UC-10 | 招待通知に応答する | [暇の共有](usecases/availability-sharing.md#uc-10-招待通知に応答する) |
| UC-11 | 無操作による自動辞退 | [暇の共有](usecases/availability-sharing.md#uc-11-無操作による自動辞退) |
| UC-12 | 今ヒマな人を見る | [暇の共有](usecases/availability-sharing.md#uc-12-今ヒマな人を見る) |
| UC-13 | 暇の共有を取り消す | [暇の共有](usecases/availability-sharing.md#uc-13-暇の共有を取り消す) |
| UC-14 | 通知を配信する | [通知配信](usecases/notification-delivery.md#uc-14-通知を配信する) |
| UC-15 | 無効になった端末トークンを失効させる | [通知配信](usecases/notification-delivery.md#uc-15-無効になった端末トークンを失効させる) |

## 登場人物

| 記号 | 役割 |
|------|------|
| **iOS** | SwiftUI クライアント。Firebase Auth でサインインし、APNs の通知を受ける |
| **API** | 本リポジトリの Go サーバー。HTTP リクエストを処理する |
| **Worker** | 同一プロセス内で動く配信ワーカー。`notification_deliveries` を取り出して APNs に送る |
| **DB** | PostgreSQL 17 |
| **Firebase** | Firebase Authentication。ID トークンの検証に公開鍵を使う |
| **APNs** | Apple Push Notification service |

## 用語

| 用語 | テーブル | 意味 |
|------|---------|------|
| 暇（availability） | `availabilities` | 「これから N 分ヒマ」という共有。有効期限を持つ |
| 招待通知 | `notifications` (`kind = 'availability_invite'`) | 暇の共有をグループメンバーへ知らせる通知。参加/辞退の 2 ボタン付き |
| 応答 | `availability_responses` | 招待通知に対する参加（`join`）または辞退（`decline`） |
| 結果通知 | `notifications` (`kind = 'response_result'`) | 応答を共有元へ知らせる通知。ボタンなし |
| 配信 | `notification_deliveries` | 通知 1 件を端末 1 台へ送る単位。送信結果を保持する |

## 前提

### 認証

`GET /health` を除く全エンドポイントで認証が必要。

```
Authorization: Bearer <Firebase ID Token>
```

API は Firebase の公開鍵で ID トークンを RS256 検証し、ペイロードの `sub` を
`firebase_uid` として扱う。以降 `users.id` を引いて `current_user` とする。
検証に失敗した場合は `401` を返す。

### 識別子

API が外部に出す ID はすべて `uuid` の文字列表現。`firebase_uid` と APNs
デバイストークンは外部システムの識別子であり、内部の主キーではない。
スキーマの詳細は [テーブル設計](schema.md) を参照。

## 記法

シーケンス図は Mermaid で書いている。GitHub と Cursor でそのまま描画される。
図中の `DB` への矢印は、実際に発行される SQL の意図を示す。
