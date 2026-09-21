# 暇の共有

このアプリの中心となる機能。「これから N 分ヒマ」をグループに流し、
反応を共有元へ返すまでの一連の流れ。
[ドキュメント一覧に戻る](../README.md)

---

## 全体像

```mermaid
sequenceDiagram
    participant 共有者 as 共有者の iOS
    participant API
    participant DB
    participant Worker
    participant APNs
    participant メンバー as メンバーの iOS

    共有者->>API: POST /availabilities (UC-09)
    API->>DB: availabilities / notifications / notification_deliveries を作成
    API-->>共有者: 201 Created

    Worker->>DB: pending の配信を取り出す (UC-14)
    Worker->>APNs: 招待通知を送信
    APNs->>メンバー: HIMASOKU_INVITE カテゴリの通知

    alt 「わかる」をタップ
        メンバー->>API: POST /availabilities/{id}/responses (join) (UC-10)
    else 「今は暇じゃない」をタップ
        メンバー->>API: POST /availabilities/{id}/responses (decline)
    else 30 秒間無操作
        メンバー->>API: POST /availabilities/{id}/responses (decline, auto_timeout) (UC-11)
    end

    API->>DB: availability_responses と結果通知を作成
    API-->>メンバー: 200 OK

    Worker->>APNs: 結果通知を送信
    APNs->>共有者: 「◯◯が共感しています！」
```

---

## UC-09 暇を共有する

**アクター**: グループに在籍する利用者
**事前条件**: 対象グループに在籍している
**事後条件**: `availabilities` に行ができ、自分以外のメンバー宛の配信が `pending` で積まれる

`POST /availabilities`

```json
{ "group_id": "…", "duration_minutes": 60, "message": "渋谷なう" }
```

```mermaid
sequenceDiagram
    participant iOS
    participant API
    participant DB

    iOS->>API: POST /availabilities (group_id, duration_minutes, message)
    API->>DB: 呼び出し元が group_id に在籍中か確認

    alt 在籍していない
        API-->>iOS: 404 Not Found
    else 在籍中
        API->>DB: BEGIN
        API->>DB: INSERT INTO availabilities (group_id, user_id, duration_minutes, message)
        Note over DB: トリガーが expires_at = started_at + duration_minutes を導出
        DB-->>API: availability (id, expires_at)

        API->>DB: SELECT user_id FROM group_members WHERE group_id = $1 AND left_at IS NULL AND user_id <> $2
        DB-->>API: 宛先ユーザーの配列

        loop 宛先ごと
            API->>DB: INSERT INTO notifications (kind, availability_id, recipient_user_id, title, body, payload)
        end

        API->>DB: SELECT id, user_id FROM devices WHERE user_id = ANY($1) AND revoked_at IS NULL
        DB-->>API: 有効な端末の配列

        loop 通知 × 端末
            API->>DB: INSERT INTO notification_deliveries (notification_id, device_id, status) VALUES (..., 'pending')
        end

        API->>DB: COMMIT
        API-->>iOS: 201 Created (availability)
    end
```

### 有効期限をアプリ側で計算しない

`expires_at` は DB のトリガーが `started_at + duration_minutes` から導出する。
アプリ側は値を渡さない。

```sql
CREATE TRIGGER availabilities_set_expires_at
    BEFORE INSERT OR UPDATE OF started_at, duration_minutes ON availabilities
    FOR EACH ROW
    EXECUTE FUNCTION set_availability_expires_at();
```

当初は生成列（`GENERATED ALWAYS AS ... STORED`）で表現する方針だったが、
PostgreSQL では実現できない。`timestamptz + interval` はタイムゾーン設定に
依存するため `STABLE` 扱いで、生成列が要求する `IMMUTABLE` を満たさず
`generation expression is not immutable` で拒否される。トリガーはこの制約を
受けないので、同じ保証を得る手段としてこちらを使っている。

### 通知は宛先ごとに 1 行

`notifications` は「暇 1 件につき 1 行」ではなく「宛先 1 人につき 1 行」作る。
iOS は `notification_id` を未応答検知の待機タスク ID として使うため、
受信者ごとに一意でなければ、誰かが応答した時点で他の端末の待機まで
解除されてしまう。

### 配信は同期で送らない

API は `notification_deliveries` を `pending` で積むところまでを行い、
APNs への送信は [UC-14](notification-delivery.md#uc-14-通知を配信する) のワーカーに委ねる。

### 既存実装との差分

| 項目 | Rails | Go |
|------|-------|-----|
| エンドポイント | `POST /notifications/group/:group_id` | `POST /availabilities` |
| 暇の永続化 | なし（通知ペイロードのみ） | `availabilities` に保存 |
| 送信方式 | 端末ごとに同期送信 | `pending` を積んでワーカーが非同期送信 |
| 在籍チェック | なし | あり |
| レスポンス | `{ total_tokens, successful, failed, details }` | `201` と作成された availability |

> **クライアント影響**: レスポンス形式が変わる。既存の iOS 実装が
> `successful` / `failed` を読んでいる場合は改修が必要。同期送信をやめる以上、
> 送信結果を応答に含めることは原理的にできない。

---

## UC-10 招待通知に応答する

**アクター**: 招待通知を受け取ったメンバー
**事前条件**: 対象の暇がまだ有効（`expires_at > now()` かつ未取消）
**事後条件**: `availability_responses` に行ができ、共有元へ結果通知が積まれる

`POST /availabilities/{availability_id}/responses`

```json
{ "action": "join", "source": "user" }
```

```mermaid
sequenceDiagram
    participant メンバー as メンバーの iOS
    participant API
    participant DB

    メンバー->>API: POST /availabilities/{id}/responses (action, source)
    API->>DB: SELECT * FROM availabilities WHERE id = $1

    alt 存在しない、取り消し済み、または期限切れ
        API-->>メンバー: 404 Not Found
    else 有効
        API->>DB: 呼び出し元が availability.group_id に在籍中か確認
        alt 在籍していない
            API-->>メンバー: 404 Not Found
        else 在籍中
            API->>DB: INSERT INTO availability_responses (availability_id, user_id, action, source)

            alt 既に応答済み
                Note over DB: availability_responses_once_per_user に違反
                DB-->>API: unique violation
                API-->>メンバー: 200 OK (冪等。結果通知は再送しない)
            else 初回の応答
                API->>DB: INSERT INTO notifications (kind='response_result', recipient_user_id=共有者)
                API->>DB: INSERT INTO notification_deliveries (...) VALUES (..., 'pending')
                API-->>メンバー: 201 Created
            end
        end
    end
```

### 送信者とグループをクライアントに申告させない

応答に必要な情報はすべて `availability_id` から辿れる。共有者は
`availabilities.user_id`、グループは `availabilities.group_id` で引ける。

Rails 版は `sender_firebase_uid` と `group_id` をリクエストボディで受け取っていた。
照合先となるレコードが存在しないため、値が正しいかを検証する手段がなく、
認証さえ通れば任意の利用者になりすまして「◯◯が共感しています」という通知を
他人に送れる状態だった。暇を実体として保存する設計上の目的の 1 つがこれになる。

### 二重応答の扱い

`UNIQUE (availability_id, user_id)` で DB が弾く。これが無いと、
[UC-11](#uc-11-無操作による自動辞退) の自動辞退とユーザーの手動操作が競合したときに、
共有者へ「参加」と「辞退」の両方の通知が届く。

### 結果通知の本文

| `action` | 本文 |
|----------|------|
| `join` | `{名前}が共感しています！` |
| `decline` | `{名前}は今は忙しいみたいです😢` |

---

## UC-11 無操作による自動辞退

**アクター**: iOS クライアント（利用者の操作なし）

招待通知を受け取った iOS は 30 秒待ち、利用者が何も操作しなければ
自動的に辞退を送る。この挙動はクライアント側の実装に依存する。

```mermaid
sequenceDiagram
    participant iOS as メンバーの iOS
    participant API

    Note over iOS: 招待通知を受信し notification_id を控える
    Note over iOS: バックグラウンドタスクで 30 秒待機

    alt 30 秒以内に操作あり
        iOS->>iOS: 操作済みフラグを立てる
        Note over iOS: 待機タスクは何もせず終了 (UC-10 で送信済み)
    else 30 秒無操作
        iOS->>API: POST /availabilities/{id}/responses (decline, auto_timeout)
        API-->>iOS: 201 Created
    end
```

サーバーから見ると [UC-10](#uc-10-招待通知に応答する) と同じ経路を通る。
違いは `source` の値だけで、`user`（明示的な辞退）と `auto_timeout`（無反応）を
区別して記録する。Rails 版は両者が同じ `DECLINE_ACTION` として届いていたため、
「断られた」のか「気づかれなかった」のかを後から判別できなかった。

---

## UC-12 今ヒマな人を見る

**アクター**: グループに在籍する利用者

`GET /groups/{group_id}/availabilities`

```mermaid
sequenceDiagram
    participant iOS
    participant API
    participant DB

    iOS->>API: GET /groups/{group_id}/availabilities
    API->>DB: 呼び出し元が在籍中か確認
    alt 在籍していない
        API-->>iOS: 404 Not Found
    else 在籍中
        API->>DB: SELECT * FROM availabilities WHERE group_id = $1 AND cancelled_at IS NULL AND expires_at > now() ORDER BY expires_at DESC
        DB-->>API: 有効な暇の配列
        API->>DB: 各 availability の応答数を集計
        DB-->>API: join / decline の件数
        API-->>iOS: 200 OK (availabilities)
    end
```

`availabilities_active_by_group_idx`（`(group_id, expires_at DESC) WHERE cancelled_at IS NULL`）が効く。

### 既存実装との差分

Rails 版には該当するエンドポイントが存在しない。暇の情報が
APNs のペイロードにしか存在しないため、通知を受け損ねると
その暇には二度とアクセスできなかった。通知が届かなかった端末の利用者や、
後からアプリを開いた利用者が状況を把握する手段がこれで確保される。

---

## UC-13 暇の共有を取り消す

**アクター**: 暇を共有した本人

`DELETE /availabilities/{availability_id}`

```mermaid
sequenceDiagram
    participant iOS
    participant API
    participant DB

    iOS->>API: DELETE /availabilities/{id}
    API->>DB: UPDATE availabilities SET cancelled_at = now() WHERE id = $1 AND user_id = $2 AND cancelled_at IS NULL

    alt 対象行なし (他人の暇、または取り消し済み)
        API-->>iOS: 404 Not Found
    else 更新成功
        API->>DB: DELETE FROM notification_deliveries WHERE notification_id IN (...) AND status = 'pending'
        Note over API,DB: 未送信ぶんだけ取り消す。送信済みは APNs から戻せない
        API-->>iOS: 204 No Content
    end
```

取り消しは論理削除にする。物理削除にすると、既に届いた招待通知に対する
応答が外部キー制約で失敗し、クライアントがエラーを受け取ることになる。
`cancelled_at` を打刻しておけば [UC-10](#uc-10-招待通知に応答する) が
`404` を返し、「もう終了しています」と伝えられる。

### 既存実装との差分

Rails 版に取り消しの概念はない。一度通知を送ると撤回できなかった。
