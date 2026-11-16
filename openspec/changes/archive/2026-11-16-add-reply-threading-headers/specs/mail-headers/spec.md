## ADDED Requirements

### Requirement: Message-ID 欄位
`MailMessage.MessageID` 有值時，郵件 SHALL 輸出 `Message-ID` header，格式為 `<local@domain>`。輸入可帶或不帶角括號；未提供時 MUST 不輸出 `Message-ID`（交給 SMTP server 產生）。

#### Scenario: 不帶角括號的輸入
- **WHEN** `MessageID` 為 `maintenance.42.7@example.com`
- **THEN** header 為 `Message-ID: <maintenance.42.7@example.com>`

#### Scenario: 已帶角括號的輸入
- **WHEN** `MessageID` 為 `<maintenance.42.7@example.com>`
- **THEN** header 為 `Message-ID: <maintenance.42.7@example.com>`（不重複加角括號）

#### Scenario: 未提供
- **WHEN** `MessageID` 為空字串
- **THEN** 郵件不含 `Message-ID` header

### Requirement: Message-ID 格式驗證
`MessageID`、`InReplyTo` 與 `References` 的每個元素 SHALL 在去除外層角括號後符合：恰好一個 `@`、`@` 前後皆非空、只含可列印 ASCII（0x21–0x7E）、不含 `<`、`>`、空白。不符合時 MUST 回傳包裝 `ErrInvalidMessageID` 的錯誤，且錯誤訊息包含欄位名稱與原始值。

#### Scenario: 缺少 @
- **WHEN** `MessageID` 為 `no-at-sign`
- **THEN** 回傳錯誤，`errors.Is(err, ErrInvalidMessageID)` 為 true，錯誤訊息精確為 `invalid message id: MessageID "no-at-sign"`

#### Scenario: 含空白或多個 @
- **WHEN** `InReplyTo` 為 `a b@example.com`，或 `References` 含 `a@b@example.com`
- **THEN** 回傳 `ErrInvalidMessageID`，訊息分別指出 `InReplyTo` 與 `References[<index>]`

#### Scenario: 含非 ASCII
- **WHEN** `MessageID` 為 `維護@example.com`
- **THEN** 回傳 `ErrInvalidMessageID`

### Requirement: In-Reply-To 與 References
`InReplyTo` 有值時 SHALL 輸出 `In-Reply-To: <id>`。`References` 有值時 SHALL 輸出 `References` header，各 ID 以角括號包住、以單一空白分隔，順序與輸入相同。`References` 為空但 `InReplyTo` 有值時，MUST 以 `[InReplyTo]` 作為 `References`。

#### Scenario: 只提供 InReplyTo
- **WHEN** `InReplyTo` 為 `parent@example.com`、`References` 為空
- **THEN** header 含 `In-Reply-To: <parent@example.com>` 與 `References: <parent@example.com>`

#### Scenario: 同時提供
- **WHEN** `InReplyTo` 為 `b@example.com`、`References` 為 `[a@example.com, <b@example.com>]`
- **THEN** `References: <a@example.com> <b@example.com>`

#### Scenario: 只提供 References
- **WHEN** `References` 有值、`InReplyTo` 為空
- **THEN** 輸出 `References`，不輸出 `In-Reply-To`

#### Scenario: References 過長時折行
- **WHEN** `References` 串接後的 header 行超過 78 字元
- **THEN** 在 ID 之間以 `CRLF + 空白` 折行，每個實體行不超過 78 字元（單一 ID 本身超過時除外），解析後的值等於以空白連接的 ID 清單

### Requirement: 自訂 Headers
`MailMessage.Headers` 中的每個項目 SHALL 以 `<名稱>: <值>` 輸出，名稱保持呼叫端給定的大小寫。名稱 MUST 只含 0x21–0x7E 且不含 `:`、不可為空；同一名稱以不分大小寫比較時不得重複。違反時回傳包裝 `ErrInvalidHeader` 的錯誤。

#### Scenario: 一般自訂 header
- **WHEN** `Headers` 為 `{"X-Notice-ID": "42", "Thread-Topic": "Maintenance"}`
- **THEN** 郵件含 `X-Notice-ID: 42` 與 `Thread-Topic: Maintenance`

#### Scenario: 名稱不合法
- **WHEN** `Headers` 含名稱 `Bad Name`、`Bad:Name` 或空字串
- **THEN** 回傳錯誤，`errors.Is(err, ErrInvalidHeader)` 為 true，訊息包含該名稱

#### Scenario: 名稱大小寫重複
- **WHEN** `Headers` 同時含 `X-Tag` 與 `x-tag`
- **THEN** 回傳 `ErrInvalidHeader`

### Requirement: 內建 header 保留
以下名稱（不分大小寫）SHALL 為保留名稱，出現在 `Headers` 時 MUST 回傳包裝 `ErrReservedHeader` 的錯誤：`From`、`To`、`Cc`、`Bcc`、`Subject`、`Date`、`Message-ID`、`In-Reply-To`、`References`、`MIME-Version`、`Content-Type`、`Content-Transfer-Encoding`。

#### Scenario: 設定保留 header
- **WHEN** `Headers` 含 `subject` 或 `Message-Id`
- **THEN** 回傳錯誤，`errors.Is(err, ErrReservedHeader)` 為 true，訊息精確為 `reserved header: "<呼叫端給的名稱>"`

### Requirement: 拒絕 CR/LF
`Subject`、`To` 與 `Cc` 的每個地址、`Headers` 的值、附件 `FileName`、inline 圖片 `CID` 與 `FileName` SHALL 不含 `\r` 或 `\n`；違反時 MUST 回傳包裝 `ErrInvalidHeader` 的錯誤，訊息包含欄位名稱。`NewManager` MUST 以同樣規則驗證 `smtpSender`。

#### Scenario: Subject 注入
- **WHEN** `Subject` 為 `Hello\r\nBcc: evil@example.com`
- **THEN** 回傳錯誤，`errors.Is(err, ErrInvalidHeader)` 為 true，且沒有產生任何訊息

#### Scenario: 收件者或檔名注入
- **WHEN** `To` 某個地址、`Headers` 某個值、附件 `FileName` 或 inline `CID` 含 `\n`
- **THEN** 回傳 `ErrInvalidHeader`，訊息指出對應欄位

#### Scenario: sender 含換行
- **WHEN** `WithSMTP` 的 sender 含 `\r\n`
- **THEN** `NewManager` 回傳錯誤，`errors.Is(err, ErrInvalidHeader)` 為 true

### Requirement: 附件檔名不可含雙引號
附件與 inline 圖片的 `FileName`、inline 圖片的 `CID` SHALL 不含 `"`（`FileName` 寫在引號內）以及 `<`、`>`（`CID`）；違反時回傳 `ErrInvalidHeader`。

#### Scenario: 檔名含雙引號
- **WHEN** 附件 `FileName` 為 `a".txt`
- **THEN** 回傳 `ErrInvalidHeader`

### Requirement: 固定 header 順序
郵件 header SHALL 依下列順序輸出（不存在者略過）：`From`、`To`、`Cc`、`Subject`、`Date`、`Message-ID`、`In-Reply-To`、`References`、自訂 header（依名稱不分大小寫排序）、`MIME-Version`、`Content-Type`。

#### Scenario: 順序穩定
- **WHEN** 以相同輸入（含多個自訂 header）建立兩次訊息
- **THEN** 兩次的 header 區塊（boundary 除外）完全相同，且順序符合上述規定

### Requirement: RFC 2047 編碼
`Subject` 與自訂 header 的值含非 ASCII 字元時 SHALL 以 `mime.QEncoding` 搭配 UTF-8 編碼；純 ASCII 值 MUST 原樣輸出。

#### Scenario: 中文主旨
- **WHEN** `Subject` 為 `RE: 維護完成通知`
- **THEN** 原始 header 不含非 ASCII 位元組、包含 `=?UTF-8?q?`，且以 `mime.WordDecoder` 解碼後精確等於 `RE: 維護完成通知`

#### Scenario: ASCII 主旨
- **WHEN** `Subject` 為 `Maintenance notice`
- **THEN** header 精確為 `Subject: Maintenance notice`

#### Scenario: 非 ASCII 自訂 header
- **WHEN** `Headers` 為 `{"Thread-Topic": "維護通知"}`
- **THEN** 該 header 值為 RFC 2047 encoded-word，解碼後等於 `維護通知`

### Requirement: 預設 From 與 Date
每封郵件 SHALL 輸出 `From: <smtpSender>`（即 smtpSender 原值）與 `Date`（RFC 5322 格式，`time.RFC1123Z`，取寄送當下時間）。

#### Scenario: From 與 Date
- **WHEN** smtpSender 為 `sender@example.com`，且建立訊息時的時間為 2026-11-16 09:30:00 +0800
- **THEN** header 含 `From: sender@example.com` 與 `Date: Mon, 16 Nov 2026 09:30:00 +0800`
